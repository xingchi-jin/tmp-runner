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

	"github.com/drone-runners/drone-runner-aws/command/harness"
	"github.com/drone-runners/drone-runner-aws/types"
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
	poolFile    string
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
			logger.WithError(ctx, loadEnvErr).Errorln("cannot load env file")
		}
	}

	// Read configs into memory
	loadedConfig, err := delegate.FromEnviron()
	if err != nil {
		logger.WithError(ctx, err).Errorln("load runner config failed")
	}
	if err = delegate.CheckInstallationConfig(loadedConfig); err != nil {
		logger.WithError(ctx, err).Fatal("Invalid configurations")
	}

	logger.ConfigureLogging(loadedConfig.Debug, loadedConfig.Trace)

	// Override pool file if provided as input
	if c.poolFile != "" {
		loadedConfig.VM.Pool.File = c.poolFile
	}

	// initialize system
	system, err := c.initializer(ctx, loadedConfig)
	if err != nil {
		return fmt.Errorf("encountered an error while wiring the system: %w", err)
	}

	remotelogger.Start(ctx, loadedConfig.Delegate.AccountID, loadedConfig.GetHarnessUrl(), loadedConfig.GetToken(), serviceName, loadedConfig.GetName(), loadedConfig.EnableRemoteLogging, loadedConfig.Server.Insecure)
	defer func() {
		err := logger.CloseHooks()
		if err != nil {
			logger.WithError(ctx, err).Warnln("Error while stopping remote logger")
		}
	}()

	// Setup the pool if it exists
	if loadedConfig.VM.Pool.File != "" {
		ctx = context.WithValue(ctx, types.Hosted, true)
		passwds := types.Passwords{AnkaToken: loadedConfig.VM.Password.AnkaToken, Tart: loadedConfig.VM.Password.Tart}
		_, err = harness.SetupPoolWithFile(ctx, c.poolFile, system.poolManager, passwds, loadedConfig.Delegate.Name,
			loadedConfig.VM.Pool.BusyMaxAge, loadedConfig.VM.Pool.FreeMaxAge, loadedConfig.VM.Pool.PurgerTimeMinutes, false)
		defer harness.Cleanup(false, system.poolManager, false, true)
		if err != nil {
			logger.WithError(ctx, err).Errorln("error while setting up pool")
			return fmt.Errorf("encountered an error while setting up pool: %w", err)
		}
	}

	// trap the os signal to gracefully shut down the http server.
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case val := <-s:
			logger.Infof(ctx, "Received OS Signal to exit server: %s", val)
			logRunnerResourceStats(ctx)
			system.delegate.Shutdown(ctx)
			cancel()
		case <-ctx.Done():
			logger.Errorln(ctx, "Received a done signal to exit server, this should not happen")
			logRunnerResourceStats(ctx)
		}
	}()
	defer signal.Stop(s)

	// Start Metrics endpoint handler
	system.metricsHandler.Handle()

	logger.Infoln(ctx, "Runner configurations loaded")

	runnerInfo, err := system.delegate.Register(ctx)
	if err != nil {
		logger.Errorf(ctx, "Registering Runner with Harness manager failed. Error: %v", err)
		return err
	}
	loadedConfig.UpsertDelegateID(runnerInfo.ID)
	logger.Infof(ctx, "Runner registered: %+v", *runnerInfo)

	logger.UpdateContextInHooks(map[string]string{"runnerId": runnerInfo.ID})

	defer func() {
		logger.Infoln(ctx, "Unregistering runner...")
		err = system.delegate.Unregister(context.Background())
		if err != nil {
			logger.Errorf(ctx, "Error while unregistering runner: %v", err)
		}
	}()

	var g errgroup.Group

	if loadedConfig.VM.Pool.File != "" {
		g.Go(func() error {
			<-ctx.Done()
			// delete unused instances for distributed pool
			return harness.Cleanup(false, system.poolManager, false, true)
		})
	}

	g.Go(func() error {
		return system.delegate.StartRunnerProcesses(ctx)
	})

	g.Go(func() error {
		if err := startHTTPServer(ctx, loadedConfig); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
				logger.Infoln(ctx, "Program gracefully terminated")
				return nil
			}
			logger.Errorf(ctx, "Program terminated with error: %s", err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.WithError(ctx, err).Errorln("One or more runner processes failed")
		return err
	}

	logger.Infoln(ctx, "All runner processes terminated")
	return err
}

func startHTTPServer(ctx context.Context, config *delegate.Config) error {
	logger.Infoln(ctx, "Starting HTTP server")

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

	cmd.Flag("pool", "file to seed the pool").
		StringVar(&c.poolFile)
}
