// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package daemonset

import (
	"os/exec"

	"github.com/drone/go-task/task"
)

// StatusCode represents status code of a task assignment/deletion.
type StatusCode string
type State string

const (
	StatusCodeSuccess StatusCode = "OK"
	StatusCodeFailed  StatusCode = "FAILED"

	StateSuccess State = "SUCCESS"
	StateFailure State = "FAILURE"
)

type (
	// represents a daemon set for the in-memory map of daemon sets
	DaemonSet struct {
		DaemonSetId string
		Type        string
		Config      DaemonSetOperationalConfig
		HttpSever   DaemonSetHttpServer
		Tasks       map[string]bool
	}

	DaemonSetHttpServer struct {
		Execution *exec.Cmd
		Port      int
	}

	// configuration for spawning new daemon set
	DaemonSetOperationalConfig struct {
		Cpu              float64                `json:"cpu"`
		Envs             []string               `json:"envs"`
		ExecutableConfig *task.ExecutableConfig `json:"executable_config"`
		MemoryBytes      int64                  `json:"memory_bytes"`
		Image            string                 `json:"image"`
		Repository       *task.Repository       `json:"repository"`
		Version          string                 `json:"version"`
	}

	// request for upserting (spawning) a new daemon set
	DaemonSetUpsertRequest struct {
		Config      DaemonSetOperationalConfig `json:"config"`
		DaemonSetId string                     `json:"daemon_set_id"`
		Type        string                     `json:"type"`
	}

	DaemonSetUpsertResponse struct {
		DaemonSetId string `json:"daemon_set_id"`
		State       State  `json:"state,omitempty"`
		Error       string `json:"error_message,omitempty"`
	}

	// request for runner to assign a new daemon task to a daemon set
	DaemonTaskAssignRequest struct {
		DaemonTaskId string           `json:"daemon_task_id"`
		Params       DaemonTaskParams `json:"params"`
		Type         string           `json:"type"`
	}

	DaemonTaskAssignResponse struct {
		DaemonTaskId string `json:"daemon_task_id"`
		State        State  `json:"state,omitempty"`
		Error        string `json:"error_message,omitempty"`
	}

	DaemonTaskParams struct {
		Base64Data []byte `json:"binary_data"`
	}

	// task processed by a daemon set
	DaemonTask struct {
		ID     string           `json:"id"`
		Params DaemonTaskParams `json:"params"`
	}

	// list of tasks processed by a daemon set
	DaemonTasks struct {
		Tasks []DaemonTask `json:"tasks"`
	}

	// response for daemon set API operations
	DaemonSetResponse struct {
		Status StatusCode `json:"status"`
		Error  string     `json:"error"`
	}
)
