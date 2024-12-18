// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package client

import (
	"context"
)

// Client is an interface which defines methods on interacting with a task managing system.
type Client interface {
	// Register registers the runner with the task server
	Register(ctx context.Context, r *RegisterRequest) (*RegisterResponse, error)

	// Heartbeat pings the task server to let it know that the runner is still alive
	Heartbeat(ctx context.Context, r *RegisterRequest) error

	// GetRunnerEvents gets a list of pending tasks that need to be executed for this runner
	GetRunnerEvents(ctx context.Context, delegateID string) (*RunnerEventsResponse, error)

	// Acquire tells the task server that the runner is ready to execute a task ID
	GetExecutionPayload(ctx context.Context, delegateID, delegateName, taskID string) (*RunnerAcquiredTasks, error)

	// SendStatus sends a response to the task server for a task ID
	SendStatus(ctx context.Context, delegateID, taskID string, req *TaskResponse) error

	// Unregister registers the runner with the task server
	Unregister(ctx context.Context, r *UnregisterRequest) error

	// ReconcileDaemonSets calls the reconcile endpoint in task server
	ReconcileDaemonSets(ctx context.Context, runnerId string, r *DaemonSetReconcileRequest) (*DaemonSetReconcileResponse, error)

	AcquireDaemonTasks(ctx context.Context, runnerId string, r *DaemonTaskAcquireRequest) (*RunnerAcquiredTasks, error)

	GetLoggingToken(ctx context.Context) (*AccessTokenBean, error)
}
