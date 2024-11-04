// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package server

import (
	"context"
	"errors"
	"github.com/harness/godotenv/v3"
	"github.com/harness/runner/delegateshell"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/logger/runnerlogs"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"os/signal"
)

type serverCommand struct {
	envFile string
}

func (c *serverCommand) run(*kingpin.ParseContext) error {
	// Load env file if exists
	if c.envFile != "" {
		loadEnvErr := godotenv.Load(c.envFile)
		if loadEnvErr != nil {
			logrus.WithError(loadEnvErr).Errorln("cannot load env file")
		}
	}

	// Read configs into memory
	loadedConfig, err := delegate.FromEnviron()
	if err != nil {
		logrus.WithError(err).Errorln("Load Runner config failed")
	}

	runnerlogs.SetLogrus()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	managerClient := client.NewManagerClient(loadedConfig.GetHarnessUrl(), loadedConfig.Delegate.AccountID, loadedConfig.GetToken(), loadedConfig.Server.Insecure, "")
	delegateShell := delegateshell.NewDelegateShell(&loadedConfig, managerClient)

	// trap the os signal to gracefully shut down the http server.
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	go func() {
		select {
		case val := <-s:
			logrus.Infof("Received OS Signal to exit server: %s", val)
			logRunnerResourceStats()
			delegateShell.Shutdown()
			cancel()
		case <-ctx.Done():
			logrus.Errorln("Received a done signal to exit server, this should not happen")
			logRunnerResourceStats()
		}
	}()
	defer signal.Stop(s)

	runnerInfo, err := delegateShell.Register(ctx)
	if err != nil {
		logrus.Errorf("Registering Runner with Harness manager failed. Error: %v", err)
		return err
	}
	logrus.Infoln("Runner registered", runnerInfo)

	defer func() {
		logrus.Infoln("Unregistering runner...")
		err = delegateShell.Unregister(context.Background())
		if err != nil {
			logrus.Errorf("Error while unregistering runner: %v", err)
		}
	}()

	var g errgroup.Group

	g.Go(func() error {
		return delegateShell.StartRunnerProcesses(ctx)
	})

	g.Go(func() error {
		if err := startHTTPServer(ctx, &loadedConfig); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
				logrus.Infoln("Program gracefully terminated")
				return nil
			}
			logrus.Errorf("Program terminated with error: %s", err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logrus.WithError(err).Errorln("One or more runner processes failed")
		return err
	}

	logrus.Infoln("All runner processes terminated")
	return err
}

func startHTTPServer(ctx context.Context, config *delegate.Config) error {
	logrus.Infoln("Starting HTTP server")

	serverInstance := Server{
		Addr:     config.Server.Bind,
		CAFile:   config.Server.CACertFile, // CA certificate file
		CertFile: config.Server.CertFile,   // Server certificate PEM file
		KeyFile:  config.Server.KeyFile,    // Server key file
		Insecure: config.Server.Insecure,   // Skip server certificate verification
	}

	// TODO: INIT_SCRIPT feature, how to and where to
	// run the setup checks / installation
	// if loadedConfig.Server.SkipPrepareServer {
	// 	logrus.Infoln("skipping prepare server eg install docker / git")
	// } else {
	// 	setup.PrepareSystem()
	// }
	// Start the HTTP server
	return serverInstance.Start(ctx)
}

func Register(app *kingpin.Application) {
	c := new(serverCommand)

	cmd := app.Command("server", "start the server").
		Action(c.run)

	cmd.Flag("env-file", "environment file").
		Default(".env").
		StringVar(&c.envFile)
}
