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

	"github.com/harness/runner/daemonset"
	"github.com/harness/runner/daemonset/client"
	"github.com/sirupsen/logrus"
)

type HttpServerDriver struct {
	client   *client.Client
	nextPort int
}

// New returns the daemon set task execution driver
func NewHttpServerDriver() *HttpServerDriver {
	return &HttpServerDriver{client: client.NewClient("http://localhost:"), nextPort: 9000}
}

// StartDaemonSet handles starting a daemon set process that runs as http server
func (h *HttpServerDriver) StartDaemonSet(binpath string, ds *daemonset.DaemonSet) (*daemonset.DaemonSet, error) {
	port := h.getPort()

	cmd, err := h.startProcess(ds.Config.Envs, binpath, port)
	if err != nil {
		return nil, err
	}

	// TODO: wait for daemon set to be ready before returning here
	ds.HttpSever = daemonset.DaemonSetHttpServer{Execution: cmd, Port: port}
	return ds, nil
}

// StopDaemonSet handles stopping a daemon set process that runs as http server
func (h *HttpServerDriver) StopDaemonSet(ds *daemonset.DaemonSet) error {
	return ds.HttpSever.Execution.Process.Kill()
}

// AssignDaemonTasks will handle assigning tasks to a daemon set process that runs as http server
func (h *HttpServerDriver) AssignDaemonTasks(ctx context.Context, ds *daemonset.DaemonSet, tasks *daemonset.DaemonTasks) (*daemonset.DaemonSetResponse, error) {
	dsUrl := getUrl(ds)
	return h.client.Assign(ctx, dsUrl, tasks)
}

// RemoveDaemonTasks will handle removing tasks from a daemon set process that runs as http server
func (h *HttpServerDriver) RemoveDaemonTasks(ctx context.Context, ds *daemonset.DaemonSet, taskIds *[]string) (*daemonset.DaemonSetResponse, error) {
	dsUrl := getUrl(ds)
	return h.client.Remove(ctx, dsUrl, taskIds)
}

// spawns daemon set process passing it the -port param
func (h *HttpServerDriver) startProcess(envs []string, binpath string, port int) (*exec.Cmd, error) {
	// TODO: Here we need to check if the daemon set is healthy before returning success (next PR will do it).
	// create the command to run the executable with the -port flag
	cmd := exec.Command(binpath, "-port", fmt.Sprintf("%d", port))

	// set the environment variables
	cmd.Env = append(os.Environ(), envs...)

	// start the command
	if err := cmd.Start(); err != nil {
		logrus.WithError(err).Error("error starting the command")
		return nil, err
	}
	return cmd, nil
}

// getPort returns the port where a new daemon set http server should listen
func (h *HttpServerDriver) getPort() int {
	port := h.nextPort
	h.nextPort++
	return port
}

// getUrl gets the top part of the daemon set http server's url
func getUrl(ds *daemonset.DaemonSet) string {
	return fmt.Sprintf("%d", ds.HttpSever.Port)
}
