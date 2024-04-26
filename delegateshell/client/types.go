package client

import (
	"encoding/json"

	"github.com/drone/go-task/task"
)

// StatusCode represents status code of a task.
type StatusCode string

// StatusCode enumeration.
const (
	StatusCodeSuccess StatusCode = "OK"
	StatusCodeFailed  StatusCode = "FAILED"
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

	TaskResponseV2 struct {
		ID    string     `json:"id"`
		Data  []byte     `json:"data"`
		Error string     `json:"error"`
		Type  string     `json:"type"`
		Code  StatusCode `json:"code"` // OK, FAILED
	}

	DelegateCapacity struct {
		MaxBuilds int `json:"maximumNumberOfBuilds"`
	}

	// TODO: use the definition in go-task repo

	RunnerAcquiredTasks struct {
		Requests []*task.Request `json:"requests"`
	}
)
