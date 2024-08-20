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
)

type DelegateShell struct {
	KeepAlive    *heartbeat.KeepAlive
	DelegateInfo *heartbeat.DelegateInfo
}

func Start(ctx context.Context, config *delegate.Config, managerClient *client.ManagerClient) (*DelegateShell, error) {

	// The poller needs a client that interacts with the task management system and a router to route the tasks
	keepAlive := heartbeat.New(config.Delegate.AccountID, config.Delegate.Name, config.GetTags(), managerClient)

	// Register the poller with manager
	runnerInfo, err := keepAlive.Register(ctx)
	if err != nil {
		return nil, err
	}
	return &DelegateShell{KeepAlive: keepAlive, DelegateInfo: runnerInfo}, nil
}

func (d *DelegateShell) SendHeartbeat(ctx context.Context) {
	d.KeepAlive.Heartbeat(ctx, d.DelegateInfo.ID, d.DelegateInfo.IP, d.DelegateInfo.Host)
}

func (d *DelegateShell) Unregister(ctx context.Context, runnerInfo *heartbeat.DelegateInfo, managerClient *client.ManagerClient) error {
	req := &client.UnregisterRequest{
		ID:       runnerInfo.ID,
		NG:       true,
		Type:     "DOCKER",
		HostName: runnerInfo.Host,
		IP:       runnerInfo.IP,
	}
	return managerClient.Unregister(ctx, req)
}
