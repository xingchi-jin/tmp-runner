// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package server

import (
	"context"
	"errors"
	"github.com/harness/runner/delegateshell"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
	"github.com/harness/runner/logger/runnerlogs"
	"github.com/harness/runner/router"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/signal"
	"time"
)

type serverCommand struct {
	envFile       string
	delegateshell *delegateshell.DelegateShell
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

	// trap the os signal to gracefully shutdown the http server.
	s := make(chan os.Signal, 1)
	stopChannel := make(chan struct{})
	doneChannel := make(chan struct{})
	signal.Notify(s, os.Interrupt)
	handleOSSignals(ctx, s, cancel, stopChannel, doneChannel)
	defer signal.Stop(s)

	managerClient := client.NewManagerClient(loadedConfig.Delegate.ManagerEndpoint, loadedConfig.Delegate.AccountID, loadedConfig.Delegate.DelegateToken, loadedConfig.Server.Insecure, "")

	delegateShell, err := register(ctx, &loadedConfig, managerClient)
	if err != nil {
		logrus.Errorf("Register Runner with Harness manager failed. Error: %v", err)
		return err
	}
	if delegateShell == nil || delegateShell.DelegateInfo == nil {
		logrus.Error("Register Runner with Harness manager failed. RunnerInfo is nil")
		return err
	}

	c.delegateshell = delegateShell
	runnerInfo := delegateShell.DelegateInfo

	logrus.Info("Runner registered", runnerInfo)

	var g errgroup.Group

	g.Go(func() error {
		pollForEvents(ctx, &loadedConfig, runnerInfo, managerClient, stopChannel, doneChannel)
		return nil
	})

	g.Go(func() error {
		c.sendHeartbeat(ctx)
		return nil
	})

	g.Go(func() error {
		if err := startHTTPServer(ctx, &loadedConfig); err != nil {
			if errors.Is(err, context.Canceled) {
				logrus.Infoln("Program gracefully terminated")
			} else {
				logrus.Errorf("Program terminated with error: %s", err)
			}
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logrus.WithError(err).Errorln("One or more processes failed")
		return err
	}

	// TODO create cleanup context
	err = c.unregisterRunner(context.Background(), runnerInfo, managerClient)
	if err != nil {
		logrus.Errorf("Error stopping polling tasks from Harness:  %s", err)
	}

	return err
}

func pollForEvents(ctx context.Context, c *delegate.Config, runnerInfo *heartbeat.DelegateInfo, managerClient *client.ManagerClient, stopChannel chan struct{}, doneChannel chan struct{}) {
	// Start polling for bijou events
	eventsServer := poller.New(managerClient, router.NewRouter(delegate.GetTaskContext(c, runnerInfo.ID)), c.Delegate.TaskStatusV2)
	// TODO: we don't need hb if we poll for task.
	// TODO: instead of hardcode 3, figure out better thread management
	if err := eventsServer.PollRunnerEvents(ctx, 3, runnerInfo.ID, time.Second*10, stopChannel, doneChannel); err != nil {
		logrus.WithError(err).Errorln("Error when polling task events")
	}
}

func register(ctx context.Context, config *delegate.Config, managerClient *client.ManagerClient) (*delegateshell.DelegateShell, error) {
	logrus.Info("Registering")
	return delegateshell.Start(ctx, config, managerClient)
}

func handleOSSignals(ctx context.Context, s chan os.Signal, cancel context.CancelFunc, stopChannel chan struct{}, doneChannel chan struct{}) {
	go func() {
		select {
		case val := <-s:
			logrus.Infof("received OS Signal to exit server: %s", val)
			logRunnerResourceStats()
			close(stopChannel) // Notify poller to stop acquiring new events
			logrus.Info("Notified poller to stop acquiring new tasks")
			<-doneChannel // Wait for all tasks to be processed
			logrus.Info("All tasks are completed, stopping task processor...")
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

func (c *serverCommand) unregisterRunner(ctx context.Context, runnerInfo *heartbeat.DelegateInfo, managerClient *client.ManagerClient) error {
	logrus.Info("Unregistering")
	return c.delegateshell.Unregister(ctx, runnerInfo, managerClient)
}

func Register(app *kingpin.Application) {
	c := new(serverCommand)

	cmd := app.Command("server", "start the server").
		Action(c.run)

	cmd.Flag("env-file", "environment file").
		Default(".env").
		StringVar(&c.envFile)
}

func (c *serverCommand) sendHeartbeat(ctx context.Context) {
	logrus.Info("Started sending heartbeat to manager")
	c.delegateshell.SendHeartbeat(ctx)
}
