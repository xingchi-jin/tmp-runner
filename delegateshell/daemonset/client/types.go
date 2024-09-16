// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package client

import (
	"os/exec"

	"github.com/drone/go-task/task"
)

type (
	// represents a daemon set for the in-memory map of daemon sets
	DaemonSet struct {
		DaemonSetId string
		Type        string
		Config      *DaemonSetOperationalConfig
		ServerInfo  *DaemonSetServerInfo
		Healthy     bool
	}

	DaemonSetServerInfo struct {
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

	DaemonTaskParams struct {
		Base64Data []byte `json:"binary_data"`
	}

	// task processed by a daemon set
	DaemonTask struct {
		ID     string           `json:"id"`
		Params DaemonTaskParams `json:"params,omitempty"`
	}

	// list of tasks processed by a daemon set
	DaemonTasks struct {
		Tasks []DaemonTask `json:"tasks"`
	}

	// metadata of a task processed by a daemon set
	DaemonTaskMetadata struct {
		ID string `json:"id"`
	}

	// list of metadata for tasks processed by a daemon set
	DaemonTasksMetadata []DaemonTaskMetadata

	// response for daemon set API operations
	DaemonSetResponse struct {
		TasksMetadata DaemonTasksMetadata `json:"tasks_metadata"`
		Error         string              `json:"error"`
	}
)
