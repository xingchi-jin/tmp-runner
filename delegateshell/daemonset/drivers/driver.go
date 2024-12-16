// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package drivers

import (
	"context"

	"github.com/harness/runner/delegateshell/daemonset/client"
)

type DaemonSetDriver interface {
	// StartDaemonSet handles starting a daemon set server
	StartDaemonSet(ctx context.Context, binpath string, ds *client.DaemonSet) (*client.DaemonSetServerInfo, error)

	// StopDaemonSet handles stopping a daemon set server
	StopDaemonSet(ds *client.DaemonSet) error

	// ListDaemonTasks will handle listing daemon tasks running in a daemon set server
	ListDaemonTasks(ctx context.Context, ds *client.DaemonSet) (*client.DaemonTasksMetadata, error)

	// AssignDaemonTasks will handle assigning tasks to a daemon set server
	AssignDaemonTasks(ctx context.Context, ds *client.DaemonSet, tasks *client.DaemonTasks) (*client.DaemonTasksMetadata, error)

	// RemoveDaemonTasks will handle removing tasks from a daemon set server
	RemoveDaemonTasks(ctx context.Context, ds *client.DaemonSet, taskIds *[]string) (*client.DaemonTasksMetadata, error)
}
