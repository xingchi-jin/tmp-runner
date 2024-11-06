// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package server

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/harness/runner/logger"

	"github.com/harness/runner/version"

	"github.com/docker/go-connections/tlsconfig"
	"golang.org/x/sync/errgroup"
)

// A Server defines parameters for running an HTTPS/TLS server.
type Server struct {
	Addr           string // TCP address to listen on
	Handler        http.Handler
	CAFile         string // CA certificate file
	CertFile       string // Server certificate PEM file
	KeyFile        string // Server key PEM file
	ClientCertFile string // Trusted client certificate PEM file for client authentication
	Insecure       bool   // run without TLS
}

// Start initializes a server to respond to HTTPS/TLS network requests.
func (s *Server) Start(ctx context.Context) error {
	// The default run mode is insecure, as most clients will run the delegate and
	// the docker runner on a same host.
	logger.Infof("Runner version: %s", version.Version)

	var tlsConfig *tls.Config
	if s.Insecure {
		tlsConfig = nil
		logger.Warnln("RUNNING IN INSECURE MODE")
	} else {
		tlsOptions := tlsconfig.Options{
			CAFile:             s.CAFile,
			CertFile:           s.CertFile,
			KeyFile:            s.KeyFile,
			ExclusiveRootPools: true,
		}
		tlsOptions.ClientAuth = tls.RequireAndVerifyClientCert
		var err error
		tlsConfig, err = tlsconfig.Server(tlsOptions)
		if err != nil {
			return err
		}
		tlsConfig.MinVersion = tls.VersionTLS13
	}

	srv := &http.Server{
		Addr:      s.Addr,
		Handler:   s.Handler,
		TLSConfig: tlsConfig,
	}

	var g errgroup.Group
	g.Go(func() error {
		// The default run mode is insecure, as most clients will run the delegate and
		// the docker runner on a same host.
		if s.Insecure {
			return srv.ListenAndServe()
		}
		return srv.ListenAndServeTLS(s.CertFile, s.KeyFile)
	})
	g.Go(func() error {
		<-ctx.Done()
		srv.Shutdown(ctx) // nolint: errcheck
		return nil
	})
	return g.Wait()
}
