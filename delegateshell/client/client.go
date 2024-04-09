// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package client

import (
	"context"
	"encoding/json"

	"github.com/drone/go-task/task"
)

// TODO: Make the structs more generic and remove Harness specific stuff
type (
	// Taken from existing manager API
	RegisterRequest struct {
		AccountID         string   `json:"accountId,omitempty"`
		RunnerName        string   `json:"delegateName,omitempty"`
		LastHeartbeat     int64    `json:"lastHeartBeat,omitempty"`
		ID                string   `json:"delegateId,omitempty"`
		Type              string   `json:"delegateType,omitempty"`
		NG                bool     `json:"ng,omitempty"`
		Polling           bool     `json:"pollingModeEnabled,omitempty"` // why Runner needs type ?? maybe should remove
		HostName          string   `json:"hostName,omitempty"`
		Connected         bool     `json:"connected,omitempty"`
		KeepAlivePacket   bool     `json:"keepAlivePacket,omitempty"`
		IP                string   `json:"ip,omitempty"`
		Tags              []string `json:"tags,omitempty"`
		HeartbeatAsObject bool     `json:"heartbeatAsObject,omitempty"` // TODO: legacy to remove
		Version           string   `json:"version,omitempty"`
	}

	// Used in the java codebase :'(
	RegisterResponse struct {
		Resource RegistrationData `json:"resource"`
	}

	RegistrationData struct {
		DelegateID string `json:"delegateId"`
	}

	TaskEventsResponse struct {
		TaskEvents []*TaskEvent `json:"delegateTaskEvents"`
	}

	RunnerEvent struct {
		AccountID  string `json:"accountId"`
		TaskID     string `json:"taskId"`
		RunnerType string `json:"runnerType"`
		TaskType   string `json:"taskType"`
	}

	RunnerEventsResponse struct {
		RunnerEvents []*RunnerEvent `json:"delegateRunnerEvents"`
	}

	TaskEvent struct {
		AccountID string `json:"accountId"`
		TaskID    string `json:"delegateTaskId"`
		Sync      bool   `json:"sync"`
		TaskType  string `json:"taskType"`
	}

	Task struct {
		ID           string          `json:"id"`
		Type         string          `json:"type"`
		Data         json.RawMessage `json:"data"`
		Async        bool            `json:"async"`
		Timeout      int             `json:"timeout"`
		Logging      LogInfo         `json:"logging"`
		DelegateInfo DelegateInfo    `json:"delegate"`
		Capabilities json.RawMessage `json:"capabilities"`
	}

	LogInfo struct {
		Token        string            `json:"token"`
		Abstractions map[string]string `json:"abstractions"`
	}

	DelegateInfo struct {
		ID         string `json:"id"`
		InstanceID string `json:"instance_id"`
		Token      string `json:"token"`
	}

	TaskResponse struct {
		ID   string          `json:"id"`
		Data json.RawMessage `json:"data"`
		Type string          `json:"type"`
		Code string          `json:"code"` // OK, FAILED, RETRY_ON_OTHER_DELEGATE
	}

	DelegateCapacity struct {
		MaxBuilds int `json:"maximumNumberOfBuilds"`
	}

	// TODO: use the definition in go-task repo

	RunnerAcquiredTasks struct {
		Requests []*task.Request `json:"requests"`
	}
)

// Client is an interface which defines methods on interacting with a task managing system.
type Client interface {
	// Register registers the runner with the task server
	Register(ctx context.Context, r *RegisterRequest) (*RegisterResponse, error)

	// Heartbeat pings the task server to let it know that the runner is still alive
	Heartbeat(ctx context.Context, r *RegisterRequest) error

	// GetTaskEvents gets a list of pending tasks that need to be executed for this runner
	GetRunnerEvents(ctx context.Context, delegateID string) (*RunnerEventsResponse, error)

	// Acquire tells the task server that the runner is ready to execute a task ID
	GetExecutionPayload(ctx context.Context, delegateID, taskID string) (*RunnerAcquiredTasks, error)

	// SendStatus sends a response to the task server for a task ID
	SendStatus(ctx context.Context, delegateID, taskID string, req *TaskResponse) error
}
