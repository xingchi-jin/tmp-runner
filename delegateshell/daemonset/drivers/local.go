// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package drivers

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/harness/runner/logger"

	"github.com/harness/runner/delegateshell/delegate"

	"github.com/harness/runner/delegateshell/daemonset/client"
)

// LocalDriver implements the `DaemonSetDriver` interface
// for daemon sets that are started as local processes
type LocalDriver struct {
	client   *client.Client
	nextPort int
	config   delegate.Config
}

// New returns the daemon set task execution driver
func NewLocalDriver() *LocalDriver {
	return &LocalDriver{client: client.NewClient("http://localhost:"), nextPort: 9000}
}

func (l *LocalDriver) StartDaemonSet(ctx context.Context, binpath string, ds *client.DaemonSet) (*client.DaemonSetServerInfo, error) {
	port := l.getPort()

	cmd, err := startProcess(ctx, ds.Config.Envs, binpath, port)
	if err != nil {
		return nil, err
	}

	// TODO: wait for daemon set to be ready before returning here
	return &client.DaemonSetServerInfo{Execution: cmd, Port: port}, nil
}

func (l *LocalDriver) StopDaemonSet(ds *client.DaemonSet) error {
	if ds.ServerInfo == nil {
		return nil
	}
	return ds.ServerInfo.Execution.Process.Kill()
}

func (l *LocalDriver) ListDaemonTasks(ctx context.Context, ds *client.DaemonSet) (*client.DaemonTasksMetadata, error) {
	dsUrl, err := getUrl(ds)
	if err != nil {
		return nil, err
	}
	resp, err := l.client.GetTasks(ctx, dsUrl)
	if err != nil || resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}
	return &resp.TasksMetadata, nil
}

func (l *LocalDriver) AssignDaemonTasks(ctx context.Context, ds *client.DaemonSet, tasks *client.DaemonTasks) (*client.DaemonTasksMetadata, error) {
	dsUrl, err := getUrl(ds)
	if err != nil {
		return nil, err
	}
	resp, err := l.client.Assign(ctx, dsUrl, tasks)
	if err != nil || resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}
	return &resp.TasksMetadata, nil
}

func (l *LocalDriver) RemoveDaemonTasks(ctx context.Context, ds *client.DaemonSet, taskIds *[]string) (*client.DaemonTasksMetadata, error) {
	dsUrl, err := getUrl(ds)
	if err != nil {
		return nil, err
	}
	resp, err := l.client.Remove(ctx, dsUrl, taskIds)
	if err != nil || resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}
	return &resp.TasksMetadata, nil
}

// getPort returns the port where a new daemon set http server should listen
func (l *LocalDriver) getPort() int {
	port := l.nextPort
	l.nextPort++
	return port
}

// spawns daemon set process passing it the DAEMON_SERVER_PORT environment variable
func startProcess(ctx context.Context, envs []string, binpath string, port int) (*exec.Cmd, error) {
	cmd := exec.Command(binpath)

	// set the environment variables
	envs = append(envs, fmt.Sprintf("DAEMON_SERVER_PORT=%d", port))
	cmd.Env = append(os.Environ(), envs...)

	// start the command
	if err := cmd.Start(); err != nil {
		logger.WithError(ctx, err).Error("error starting the command")
		return nil, err
	}
	return cmd, nil
}

// getUrl gets the top part of the daemon set http server's url
func getUrl(ds *client.DaemonSet) (string, error) {
	if ds.ServerInfo == nil {
		return "", fmt.Errorf("no ServerInfo in daemon set")
	}
	return fmt.Sprintf("%d", ds.ServerInfo.Port), nil
}
