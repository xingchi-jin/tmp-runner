// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package server

import (
	"context"
	"os"
	"os/signal"

	"github.com/harness/runner/delegateshell"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/router"
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

	// create the http serverInstance.
	serverInstance := Server{
		Addr:     loadedConfig.Server.Bind,
		CAFile:   loadedConfig.Server.CACertFile, // CA certificate file
		CertFile: loadedConfig.Server.CertFile,   // Server certificate PEM file
		KeyFile:  loadedConfig.Server.KeyFile,    // Server key file
		Insecure: loadedConfig.Server.Insecure,   // Skip server certificate verification
	}

	// trap the os signal to gracefully shutdown the http server.
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	defer func() {
		signal.Stop(s)
		cancel()
	}()
	go func() {
		select {
		case val := <-s:
			logrus.Infof("received OS Signal to exit server: %s", val)
			cancel()
		case <-ctx.Done():
			logrus.Infoln("received a done signal to exit server")
		}
	}()

	logrus.Info("registering")

	err = startHarnessTasks(ctx, &loadedConfig)
	if err != nil {
		cancel()
		logrus.Error("Error starting polling tasks from Harness")
	}

	// TODO: INIT_SCRIPT feature, how to and where to
	// run the setup checks / installation
	// if loadedConfig.Server.SkipPrepareServer {
	// 	logrus.Infoln("skipping prepare server eg install docker / git")
	// } else {
	// 	setup.PrepareSystem()
	// }
	// starts the http server.
	err = serverInstance.Start(ctx)
	if err == context.Canceled {
		logrus.Infoln("program gracefully terminated")
		return nil
	}

	if err != nil {
		logrus.Errorf("program terminated with error: %s", err)
	}

	return err
}

func Register(app *kingpin.Application) {
	c := new(serverCommand)

	cmd := app.Command("server", "start the server").
		Action(c.run)

	cmd.Flag("env-file", "environment file").
		Default(".env").
		StringVar(&c.envFile)
}

func startHarnessTasks(ctx context.Context, config *delegate.Config) error {
	_, err := delegateshell.Start(ctx, config, router.NewRouter())
	return err
}
