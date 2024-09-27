package daemontask

import (
	"context"
	"encoding/json"

	"github.com/drone/go-task/task"
	"github.com/harness/runner/delegateshell/daemonset"
	"github.com/harness/runner/delegateshell/daemonset/client"
	"github.com/sirupsen/logrus"
)

type State string

const (
	StateSuccess State = "SUCCESS"
	StateFailure State = "FAILURE"
)

type (
	// request for upserting (spawning) a new daemon set
	DaemonSetUpsertRequest struct {
		Config      client.DaemonSetOperationalConfig `json:"config"`
		DaemonSetId string                            `json:"daemon_set_id"`
		Type        string                            `json:"type"`
	}

	DaemonSetUpsertResponse struct {
		DaemonSetId string `json:"daemon_set_id"`
		State       State  `json:"state,omitempty"`
		Error       string `json:"error_message,omitempty"`
	}

	// request for runner to assign a new daemon task to a daemon set
	DaemonTaskAssignRequest struct {
		DaemonTaskId string                  `json:"daemon_task_id"`
		Params       client.DaemonTaskParams `json:"params"`
		Type         string                  `json:"type"`
	}

	DaemonTaskAssignResponse struct {
		DaemonTaskId string `json:"daemon_task_id"`
		State        State  `json:"state,omitempty"`
		Error        string `json:"error_message,omitempty"`
	}
)

type DaemonSetTaskHandler struct {
	daemonSetManager *daemonset.DaemonSetManager
}

func NewDaemonSetTaskHandler(daemonSetManager *daemonset.DaemonSetManager) *DaemonSetTaskHandler {
	return &DaemonSetTaskHandler{
		daemonSetManager: daemonSetManager,
	}
}

// HandleUpsert handles runner tasks for upserting a daemon set
func (d *DaemonSetTaskHandler) HandleUpsert(ctx context.Context, req *task.Request) task.Response {
	spec := new(DaemonSetUpsertRequest)
	err := json.Unmarshal(req.Task.Data, spec)
	if err != nil {
		return task.Error(err)
	}

	logrus.Infof("Attempting to upsert daemo nset with following configs: %+v, config: %+v", spec, spec.Config)
	_, err = d.daemonSetManager.UpsertDaemonSet(ctx, spec.DaemonSetId, spec.Type, &spec.Config)
	if err != nil {
		return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: spec.DaemonSetId, State: StateFailure, Error: err.Error()})
	}
	return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: spec.DaemonSetId, State: StateSuccess})
}

// HandleTaskAssign handles runner tasks for assigning a new daemon task to a daemon set
func (d *DaemonSetTaskHandler) HandleTaskAssign(ctx context.Context, req *task.Request) task.Response {
	spec := new(DaemonTaskAssignRequest)
	err := json.Unmarshal(req.Task.Data, spec)
	if err != nil {
		return task.Error(err)
	}

	logrus.Infof("Received daemon task assign requests: taskId: %s, params: %s, secrets: %+v", spec.DaemonTaskId, spec.Params, req.Secrets)

	daemonTask := client.DaemonTask{ID: spec.DaemonTaskId, Params: spec.Params, Secrets: req.Secrets}
	err = d.daemonSetManager.AssignDaemonTasks(ctx, spec.Type, &client.DaemonTasks{Tasks: []client.DaemonTask{daemonTask}})
	if err != nil {
		return task.Respond(&DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: StateFailure, Error: err.Error()})
	}
	return task.Respond(&DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: StateSuccess})
}
