// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package server

import (
	"context"
	"errors"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/heartbeat"
	"os"
	"os/signal"

	"github.com/harness/runner/delegateshell"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/logger/runnerlogs"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
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

	// trap the os signal to gracefully shutdown the http server.
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	handleOSSignals(ctx, s, cancel)
	defer func() {
		signal.Stop(s)
		cancel()
	}()

	// Create a manager client
	managerClient := client.NewManagerClient(loadedConfig.Delegate.ManagerEndpoint, loadedConfig.Delegate.AccountID, loadedConfig.Delegate.DelegateToken, loadedConfig.Server.Insecure, "")

	runnerInfo, err := startHarnessTasks(ctx, &loadedConfig, managerClient)
	if err != nil {
		logrus.Error("Error starting polling tasks from Harness")
		cancel()
	}
	defer func(ctx context.Context, runnerInfo *heartbeat.DelegateInfo, managerClient *client.ManagerClient) {
		err := stopHarnessTasks(ctx, runnerInfo, managerClient)
		if err != nil {
			logrus.Error("Error stopping polling tasks from Harness")
		}
	}(ctx, runnerInfo, managerClient)

	// starts the http server.
	if err := startHTTPServer(ctx, &loadedConfig); err != nil {
		if errors.Is(err, context.Canceled) {
			logrus.Infoln("Program gracefully terminated")
			return nil
		}
		logrus.Errorf("Program terminated with error: %s", err)
		return err
	}

	return err
}

func startHarnessTasks(ctx context.Context, config *delegate.Config, managerClient *client.ManagerClient) (*heartbeat.DelegateInfo, error) {
	logrus.Info("Registering")
	runnerInfo, err := delegateshell.Start(ctx, config, managerClient)
	return runnerInfo, err
}

func handleOSSignals(ctx context.Context, s chan os.Signal, cancel context.CancelFunc) {
	go func() {
		select {
		case val := <-s:
			logrus.Infof("received OS Signal to exit server: %s", val)
			logRunnerResourceStats()
			cancel()
		case <-ctx.Done():
			logrus.Infoln("received a done signal to exit server")
			logRunnerResourceStats()
		}
	}()
}

func startHTTPServer(ctx context.Context, config *delegate.Config) error {
	logrus.Info("Starting HTTP server")

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
	err := serverInstance.Start(ctx)
	if errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}

func stopHarnessTasks(ctx context.Context, runnerInfo *heartbeat.DelegateInfo, managerClient *client.ManagerClient) error {
	logrus.Info("Unregistering")
	return delegateshell.Shutdown(ctx, runnerInfo, managerClient)
}

func Register(app *kingpin.Application) {
	c := new(serverCommand)

	cmd := app.Command("server", "start the server").
		Action(c.run)

	cmd.Flag("env-file", "environment file").
		Default(".env").
		StringVar(&c.envFile)
}
