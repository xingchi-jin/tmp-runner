// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/harness/runner/logger/remotelogger"

	"github.com/harness/runner/logger"

	"github.com/harness/godotenv/v3"
	"github.com/harness/runner/delegateshell/delegate"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"
)

const serviceName = "runner"

type serverCommand struct {
	envFile     string
	initializer func(context.Context, *delegate.Config) (*System, error)
}

func (c *serverCommand) run(*kingpin.ParseContext) error {
	// Create context that listens for the interrupt signal from the OS.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load env file if exists
	if c.envFile != "" {
		loadEnvErr := godotenv.Load(c.envFile)
		if loadEnvErr != nil {
			logger.WithError(loadEnvErr).Errorln("cannot load env file")
		}
	}

	// Read configs into memory
	loadedConfig, err := delegate.FromEnviron()
	if err != nil {
		logger.WithError(err).Errorln("load runner config failed")
	}

	logger.ConfigureLogging(loadedConfig.Debug, loadedConfig.Trace)

	// initialize system
	system, err := c.initializer(ctx, loadedConfig)
	if err != nil {
		return fmt.Errorf("encountered an error while wiring the system: %w", err)
	}

	remotelogger.Start(ctx, loadedConfig.Delegate.AccountID, loadedConfig.GetHarnessUrl(), loadedConfig.GetToken(), serviceName, loadedConfig.GetName(), loadedConfig.EnableRemoteLogging, loadedConfig.Server.Insecure)
	defer func() {
		err := logger.CloseHooks()
		if err != nil {
			logger.WithError(err).Warnln("Error while stopping remote logger")
		}
	}()

	// trap the os signal to gracefully shut down the http server.
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case val := <-s:
			logger.Infof("Received OS Signal to exit server: %s", val)
			logRunnerResourceStats()
			system.delegate.Shutdown()
			cancel()
		case <-ctx.Done():
			logger.Errorln("Received a done signal to exit server, this should not happen")
			logRunnerResourceStats()
		}
	}()
	defer signal.Stop(s)

	runnerInfo, err := system.delegate.Register(ctx)
	if err != nil {
		logger.Errorf("Registering Runner with Harness manager failed. Error: %v", err)
		return err
	}
	loadedConfig.UpsertDelegateID(runnerInfo.ID)
	logger.Infoln("Runner registered", runnerInfo)

	logger.UpdateContextInHooks(map[string]string{"runnerId": runnerInfo.ID})

	defer func() {
		logger.Infoln("Unregistering runner...")
		err = system.delegate.Unregister(context.Background())
		if err != nil {
			logger.Errorf("Error while unregistering runner: %v", err)
		}
	}()

	var g errgroup.Group

	g.Go(func() error {
		return system.delegate.StartRunnerProcesses(ctx)
	})

	g.Go(func() error {
		if err := startHTTPServer(ctx, loadedConfig); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
				logger.Infoln("Program gracefully terminated")
				return nil
			}
			logger.Errorf("Program terminated with error: %s", err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.WithError(err).Errorln("One or more runner processes failed")
		return err
	}

	logger.Infoln("All runner processes terminated")
	return err
}

func startHTTPServer(ctx context.Context, config *delegate.Config) error {
	logger.Infoln("Starting HTTP server")

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
	// 	logger.Infoln("skipping prepare server eg install docker / git")
	// } else {
	// 	setup.PrepareSystem()
	// }
	// Start the HTTP server
	return serverInstance.Start(ctx)
}

func Register(app *kingpin.Application, initializer func(context.Context, *delegate.Config) (*System, error)) {
	c := new(serverCommand)
	c.initializer = initializer

	cmd := app.Command("server", "start the server").
		Action(c.run)

	cmd.Flag("env-file", "environment file").
		Default(".env").
		StringVar(&c.envFile)
}
