package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/harness/runner/logger"

	"github.com/cenkalti/backoff"
)

// An HTTPClient manages communication with the runner API.
type HTTPClient struct {
	Client     *http.Client
	Endpoint   string
	SkipVerify bool
}

// defaultClient is the default http.Client.
var defaultClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// New returns a new client.
func New(endpoint string, skipverify bool, additionalCertsDir string) *HTTPClient {
	return getClient(endpoint, skipverify, additionalCertsDir)
}

func getClient(endpoint string, skipverify bool, additionalCertsDir string) *HTTPClient {
	c := &HTTPClient{
		Endpoint:   endpoint,
		SkipVerify: skipverify,
		Client:     defaultClient,
	}
	if skipverify {
		c.Client = &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: skipverify, //nolint:gosec
				},
			},
		}
	} else if additionalCertsDir != "" {
		// If additional certs are specified, we append them to the existing cert chain

		// Use the system certs if possible
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		logger.Infof("additional certs dir to allow: %s\n", additionalCertsDir)

		files, err := os.ReadDir(additionalCertsDir)
		if err != nil {
			logger.Errorf("could not read directory %s, error: %s", additionalCertsDir, err)
			c.Client = clientWithRootCAs(skipverify, rootCAs)
			return c
		}

		// Go through all certs in this directory and add them to the global certs
		for _, f := range files {
			path := filepath.Join(additionalCertsDir, f.Name())
			logger.Infof("trying to add certs at: %s to root certs\n", path)
			// Create TLS config using cert PEM
			rootPem, err := os.ReadFile(path)
			if err != nil {
				logger.Errorf("could not read certificate file (%s), error: %s", path, err.Error())
				continue
			}
			// Append certs to the global certs
			ok := rootCAs.AppendCertsFromPEM(rootPem)
			if !ok {
				logger.Errorf("error adding cert (%s) to pool, please check format of the certs provided.", path)
				continue
			}
			logger.Infof("successfully added cert at: %s to root certs", path)
		}
		c.Client = clientWithRootCAs(skipverify, rootCAs)
	}
	return c
}

func clientWithRootCAs(skipverify bool, rootCAs *x509.CertPool) *http.Client {
	// Create the HTTP Client with certs
	config := &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: skipverify,
	}
	if rootCAs != nil {
		config.RootCAs = rootCAs
	}
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: config,
		},
	}
}

// do is a helper function that posts a signed http request with
// the input encoded and response decoded from json.
func (p *HTTPClient) Do(ctx context.Context, path, method string, headers map[string]string, in *bytes.Buffer) (*http.Response, []byte, error) {
	endpoint := p.Endpoint + path
	req, err := http.NewRequest(method, endpoint, in)
	if err != nil {
		return nil, nil, err
	}
	req = req.WithContext(ctx)

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	res, err := p.Client.Do(req)
	if res != nil {
		defer func() {
			// drain the response body so we can reuse
			// this connection.
			if _, err = io.Copy(io.Discard, io.LimitReader(res.Body, 4096)); err != nil {
				logger.Errorf("could not drain response body: %s", err)
			}
			res.Body.Close()
		}()
	}
	if err != nil {
		return res, nil, err
	}

	// if the response body return no content we exit
	// immediately. We do not read or unmarshal the response
	// and we do not return an error.
	if res.StatusCode == 204 {
		return res, nil, nil
	}

	// else read the response body into a byte slice.
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res, nil, err
	}

	if res.StatusCode > 299 {
		// if the response body includes an error message
		// we should return the error string.
		if len(body) != 0 {
			return res, body, errors.New(
				string(body),
			)
		}
		// if the response body is empty we should return
		// the default status code text.
		return res, body, errors.New(
			http.StatusText(res.StatusCode),
		)
	}
	return res, body, nil
}

func createBackoff(ctx context.Context, maxElapsedTime time.Duration) backoff.BackOffContext {
	exp := backoff.NewExponentialBackOff()
	exp.MaxElapsedTime = maxElapsedTime
	return backoff.WithContext(exp, ctx)
}
