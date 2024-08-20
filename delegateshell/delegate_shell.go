// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegateshell

import (
	"context"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
	"github.com/harness/runner/router"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"time"
)

type DelegateShell struct {
	Info          *heartbeat.DelegateInfo
	Config        *delegate.Config
	ManagerClient *client.ManagerClient
	KeepAlive     *heartbeat.KeepAlive
}

func NewDelegateShell(config *delegate.Config, managerClient *client.ManagerClient) *DelegateShell {

	// The poller needs a client that interacts with the task management system and a router to route the tasks
	keepAlive := heartbeat.New(config.Delegate.AccountID, config.Delegate.Name, config.GetTags(), managerClient)
	return &DelegateShell{
		Config:        config,
		KeepAlive:     keepAlive,
		ManagerClient: managerClient,
	}
}

func (d *DelegateShell) Register(ctx context.Context) (*heartbeat.DelegateInfo, error) {
	logrus.Info("Registering runner")
	// Register the poller with manager
	runnerInfo, err := d.KeepAlive.Register(ctx)
	if err != nil {
		return nil, err
	}
	d.Info = runnerInfo
	return runnerInfo, nil
}

func (d *DelegateShell) Unregister(ctx context.Context, runnerInfo *heartbeat.DelegateInfo) error {
	req := &client.UnregisterRequest{
		ID:       runnerInfo.ID,
		NG:       true,
		Type:     "DOCKER",
		HostName: runnerInfo.Host,
		IP:       runnerInfo.IP,
	}
	return d.ManagerClient.Unregister(ctx, req)
}

func (d *DelegateShell) StartRunnerProcesses(ctx context.Context, stopChannel chan struct{}, doneChannel chan struct{}) error {
	var rg errgroup.Group

	rg.Go(func() error {
		return d.startPoller(ctx, stopChannel, doneChannel)
	})

	rg.Go(func() error {
		return d.sendHeartbeat(ctx)
	})
	return rg.Wait()
}

func (d *DelegateShell) sendHeartbeat(ctx context.Context) error {

	logrus.Info("Starting sending heartbeat to manager...")
	d.KeepAlive.Heartbeat(ctx, d.Info.ID, d.Info.IP, d.Info.Host)
	return nil
}

func (d *DelegateShell) startPoller(ctx context.Context, stopChannel chan struct{}, doneChannel chan struct{}) error {
	// Start polling for bijou events
	eventsServer := poller.New(d.ManagerClient, router.NewRouter(delegate.GetTaskContext(d.Config, d.Info.ID)), d.Config.Delegate.TaskStatusV2)
	// TODO: we don't need hb if we poll for task.
	// TODO: instead of hardcode 3, figure out better thread management
	if err := eventsServer.PollRunnerEvents(ctx, 3, d.Info.ID, time.Second*10, stopChannel, doneChannel); err != nil {
		logrus.WithError(err).Errorln("Error when polling task events")
		return err
	}
	return nil
}
