// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegateshell

import (
	"context"
	"time"

	"github.com/harness/runner/logger"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/downloader"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/daemonset"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
	"golang.org/x/sync/errgroup"
)

type DelegateShell struct {
	Info                *heartbeat.DelegateInfo
	Config              *delegate.Config
	ManagerClient       client.Client
	KeepAlive           *heartbeat.KeepAlive
	Poller              *poller.Poller
	Downloader          downloader.Downloader
	DaemonSetManager    *daemonset.DaemonSetManager
	DaemonSetReconciler *daemonset.DaemonSetReconciler
	Router              *task.Router
}

func NewDelegateShell(
	config *delegate.Config,
	managerClient client.Client,
	router *task.Router,
	daemonSetManager *daemonset.DaemonSetManager,
	daemonSetReconciler *daemonset.DaemonSetReconciler,
	downloader downloader.Downloader,
	poller *poller.Poller,
	keepAlive *heartbeat.KeepAlive,
) *DelegateShell {
	return &DelegateShell{
		Config:              config,
		KeepAlive:           keepAlive,
		ManagerClient:       managerClient,
		Downloader:          downloader,
		Router:              router,
		Poller:              poller,
		DaemonSetManager:    daemonSetManager,
		DaemonSetReconciler: daemonSetReconciler,
	}
}

func (d *DelegateShell) Register(ctx context.Context) (*heartbeat.DelegateInfo, error) {
	logger.Infoln(ctx, "Registering runner")
	// Register the poller with manager
	runnerInfo, err := d.KeepAlive.Register(ctx)
	if err != nil {
		return nil, err
	}
	d.Info = runnerInfo
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
	logger.Infoln(ctx, "Started sending heartbeat to manager...")
	d.KeepAlive.Heartbeat(ctx, d.Info.ID, d.Info.IP, d.Info.Host)
	return nil
}

func (d *DelegateShell) startDaemonSetReconcile(ctx context.Context) error {
	if err := d.DaemonSetReconciler.Start(ctx, d.Info.ID, time.Minute*1); err != nil {
		logger.WithError(ctx, err).Errorln("Error starting reconcile for daemon sets")
		return err
	}
	return nil
}

func (d *DelegateShell) startPoller(ctx context.Context) error {
	if err := d.Poller.PollRunnerEvents(ctx, d.Config.Delegate.ParallelWorkers, d.Info.ID, d.Info.Name, time.Duration(d.Config.Delegate.PollIntervalMilliSecs)*time.Millisecond); err != nil {
		logger.WithError(ctx, err).Errorln("Error when polling task events")
		return err
	}
	return nil
}

func (d *DelegateShell) Shutdown(ctx context.Context) {
	d.DaemonSetReconciler.Stop(ctx)
	d.Poller.Shutdown(ctx)
	d.DaemonSetManager.RemoveAllDaemonSets(ctx)
}
