// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegateshell

import (
	"context"
	"time"

	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
	"github.com/harness/runner/router"
	"github.com/sirupsen/logrus"
)

func Start(ctx context.Context, config *delegate.Config) (*heartbeat.DelegateInfo, error) {
	// Create a delegate client
	managerClient := client.NewManagerClient(config.Delegate.ManagerEndpoint, config.Delegate.AccountID, config.Delegate.DelegateToken, config.Server.Insecure, "")

	// The poller needs a client that interacts with the task management system and a router to route the tasks
	keepAlive := heartbeat.New(config.Delegate.AccountID, config.Delegate.Name, config.GetTags(), managerClient)

	// Register the poller
	info, err := keepAlive.Register(ctx)
	if err != nil {
		logrus.WithError(err).Errorln("Register Runner with Harness manager failed.")
		return info, err
	}

	logrus.Info("Runner registered", info)
	// Start polling for bijou events
	eventsServer := poller.New(managerClient, router.NewRouter(delegate.GetTaskContext(config, info.ID)), config.Delegate.TaskStatusV2)
	// TODO: we don't need hb if we poll for task.
	// TODO: instead of hardcode 3, figure out better thread management
	if err = eventsServer.PollRunnerEvents(ctx, 3, info.ID, time.Second*10); err != nil {
		logrus.WithError(err).Errorln("Error when polling task events")
		return info, err
	}
	return info, nil
}
