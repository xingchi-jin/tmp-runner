// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegateshell

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/drone/go-task/task/cloner"
	"github.com/drone/go-task/task/download"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/daemonset"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
	"github.com/harness/runner/router"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type DelegateShell struct {
	Info                *heartbeat.DelegateInfo
	Config              *delegate.Config
	ManagerClient       *client.ManagerClient
	KeepAlive           *heartbeat.KeepAlive
	Poller              *poller.Poller
	Downloader          download.Downloader
	DaemonSetManager    *daemonset.DaemonSetManager
	DaemonSetReconciler *daemonset.DaemonSetReconciler
}

func NewDelegateShell(config *delegate.Config, managerClient *client.ManagerClient) *DelegateShell {

	cache, err := os.UserCacheDir()
	if err != nil {
		log.Fatalln(err)
	}
	downloader := download.New(cloner.Default(), cache)

	// The poller needs a client that interacts with the task management system and a router to route the tasks
	keepAlive := heartbeat.New(config.Delegate.AccountID, config.Delegate.Name, config.GetTags(), managerClient)
	return &DelegateShell{Config: config,
		KeepAlive:     keepAlive,
		ManagerClient: managerClient,
		Downloader:    downloader,
	}
}

func (d *DelegateShell) Register(ctx context.Context) (*heartbeat.DelegateInfo, error) {
	logrus.Infoln("Registering runner")
	// Register the poller with manager
	runnerInfo, err := d.KeepAlive.Register(ctx)
	if err != nil {
		return nil, err
	}
	d.Info = runnerInfo
	d.DaemonSetManager = daemonset.NewDaemonSetManager(d.Downloader, delegate.IsK8sRunner(delegate.GetTaskContext(d.Config, d.Info.ID).RunnerType))
	d.DaemonSetReconciler = daemonset.NewDaemonSetReconciler(d.DaemonSetManager, d.ManagerClient)
	return runnerInfo, nil
}

func (d *DelegateShell) Unregister(ctx context.Context) error {
	req := &client.UnregisterRequest{
		ID:       d.Info.ID,
		NG:       true,
		Type:     "DOCKER",
		HostName: d.Info.Host,
		IP:       d.Info.IP,
	}
	return d.ManagerClient.Unregister(ctx, req)
}

func (d *DelegateShell) StartRunnerProcesses(ctx context.Context) error {
	var rg errgroup.Group

	rg.Go(func() error {
		return d.startDaemonSetReconcile(ctx)
	})

	rg.Go(func() error {
		return d.startPoller(ctx)
	})

	rg.Go(func() error {
		return d.sendHeartbeat(ctx)
	})
	return rg.Wait()
}

func (d *DelegateShell) sendHeartbeat(ctx context.Context) error {
	logrus.Infoln("Started sending heartbeat to manager...")
	d.KeepAlive.Heartbeat(ctx, d.Info.ID, d.Info.IP, d.Info.Host)
	return nil
}

func (d *DelegateShell) startDaemonSetReconcile(ctx context.Context) error {
	if err := d.DaemonSetReconciler.Start(ctx, d.Info.ID, time.Minute*1); err != nil {
		logrus.WithError(err).Errorln("Error starting reconcile for daemon sets")
		return err
	}
	return nil
}

func (d *DelegateShell) startPoller(ctx context.Context) error {
	// Start polling for bijou events
	d.Poller = poller.New(d.ManagerClient, router.NewRouter(delegate.GetTaskContext(d.Config, d.Info.ID), d.Downloader, d.DaemonSetManager), d.Config.Delegate.TaskStatusV2)
	// TODO: we don't need hb if we poll for task.
	// TODO: instead of hardcode 3, figure out better thread management
	if err := d.Poller.PollRunnerEvents(ctx, 3, d.Info.ID, d.Info.Name, time.Second*10); err != nil {
		logrus.WithError(err).Errorln("Error when polling task events")
		return err
	}
	return nil
}

func (d *DelegateShell) Shutdown() {
	d.DaemonSetReconciler.Stop()
	d.Poller.Shutdown()
}
