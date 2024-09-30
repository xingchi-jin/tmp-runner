package client

import (
	"encoding/json"

	"github.com/drone/go-task/task"
	"github.com/harness/runner/delegateshell/daemonset/client"
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
		Code string          `json:"code"` // OK, FAILED
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

	UnregisterRequest struct {
		ID        string `json:"delegateId,omitempty"`
		HostName  string `json:"hostName,omitempty"`
		NG        bool   `json:"ng,omitempty"`
		Type      string `json:"delegateType,omitempty"`
		IP        string `json:"ipAddress,omitempty"`
		OrgID     string `json:"orgIdentifier,omitempty"`
		ProjectID string `json:"projectIdentifier,omitempty"`
	}

	// Types for daemon set reconciliation flow
	DaemonSetReconcileRequest struct {
		Data []DaemonSetReconcileRequestEntry `json:"data"`
	}

	DaemonSetReconcileRequestEntry struct {
		DaemonSetId string                            `json:"daemon_set_id"`
		Type        string                            `json:"type"`
		Config      client.DaemonSetOperationalConfig `json:"config"`
		Healthy     bool                              `json:"healthy"`
	}

	DaemonSetReconcileResponse struct {
		Data []DaemonSetServerInfo `json:"data"`
	}

	DaemonSetServerInfo struct {
		DaemonSetId string                            `json:"daemon_set_id"`
		Config      client.DaemonSetOperationalConfig `json:"config"`
		SkipUpdate  bool                              `json:"skip_update"`
		TaskIds     []string                          `json:"task_ids"`
		Type        string                            `json:"type"`
	}

	DaemonTaskAcquireRequest struct {
		TaskIds []string `json:"task_ids"`
	}

	DaemonTaskAcquireResponse struct {
		Tasks []AcquiredDaemonTask `json:"tasks"`
	}

	AcquiredDaemonTask struct {
		DaemonTaskId string                  `json:"daemon_task_id"`
		Params       client.DaemonTaskParams `json:"params"`
		Type         string                  `json:"type"`
	}
)
