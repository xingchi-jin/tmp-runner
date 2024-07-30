package daemonset

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/harness/runner/utils"
)

const (
	endpoint = "%d/tasks"
)

type Client struct {
	utils.HTTPClient
}

func newClient() *Client {
	return &Client{
		HTTPClient: *utils.New("http://localhost:", true, ""),
	}
}

// Assign sends a new daemon task to a daemon set specified by the `port` argument
func (p *Client) Assign(ctx context.Context, port int, r *DaemonTasks) (*DaemonSetResponse, error) {
	req := r
	resp := &DaemonSetResponse{}
	path := fmt.Sprintf(endpoint, port)
	_, err := p.doJson(ctx, path, "POST", req, resp)
	return resp, err
}

// Remove removes a daemon task from a daemon set, specified by the `port` argument
func (p *Client) Remove(ctx context.Context, port int, r *[]string) (*DaemonSetResponse, error) {
	resp := &DaemonSetResponse{}
	path := fmt.Sprintf(endpoint, port)

	// Build the query params
	queryParams := url.Values{}
	for _, id := range *r {
		queryParams.Add("taskIds", id)
	}

	// Append query parameters to the path
	pathWithQuery := fmt.Sprintf("%s?%s", path, queryParams.Encode())
	_, err := p.doJson(ctx, pathWithQuery, "DELETE", nil, resp)
	return resp, err
}

func (p *Client) doJson(ctx context.Context, path, method string, in, out interface{}) (*http.Response, error) {
	var buf = &bytes.Buffer{}
	// marshal the input payload into json format and copy
	// to an io.ReadCloser.
	if in != nil {
		if err := json.NewEncoder(buf).Encode(in); err != nil {
			p.Logger.Errorf("could not encode input payload: %s", err)
		}
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	res, body, err := p.Do(ctx, path, method, headers, buf)
	if err != nil {
		return res, err
	}
	if nil == out {
		return res, nil
	}
	if jsonErr := json.Unmarshal(body, out); jsonErr != nil {
		return res, jsonErr
	}

	return res, nil
}
