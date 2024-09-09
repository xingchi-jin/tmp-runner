// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/download"
	"github.com/harness/runner/daemonset"
	"github.com/harness/runner/daemonset/drivers"
	"github.com/harness/runner/daemonset/utils"
	"github.com/sirupsen/logrus"
)

var (
	taskYmlPath = "task.yml"
)

type Manager struct {
	downloader download.Downloader
	// we use `sync.Map` to store running daemon sets. This is because, even though operations are atomic
	// per daemon set type (see comment on `lock` below), there can be multiple simultaneous operations
	// for different daemon set types. `sync.Map` is an appropriate date structure for this use case.
	// From https://pkg.go.dev/sync#Map: "the (sync.Map) type is optimized for two common use cases: ...
	// when multiple goroutines read, write, and overwrite entries for disjoint sets of keys. ..."
	daemonsets *sync.Map
	// the `lock` here is a wrapper for a map of locks, indexed by daemon set's type
	// so that we can make sure operations are atomic for each daemon set type
	lock   *KeyLock
	driver *drivers.HttpServerDriver
}

// New returns the daemon set task execution driver
func New(d download.Downloader, isK8s bool) *Manager {
	// TODO: Add suport for daemon sets in k8s runner
	return &Manager{downloader: d, daemonsets: &sync.Map{}, lock: NewKeyLock(), driver: drivers.NewHttpServerDriver()}
}

// HandleUpsert handles runner tasks for upserting a daemon set
func (m *Manager) HandleUpsert(ctx context.Context, req *task.Request) task.Response {
	spec := new(daemonset.DaemonSetUpsertRequest)
	err := json.Unmarshal(req.Task.Data, spec)
	if err != nil {
		return task.Error(err)
	}

	m.lock.Lock(spec.Type)
	defer m.lock.Unlock(spec.Type)

	ds := &daemonset.DaemonSet{DaemonSetId: spec.DaemonSetId, Type: spec.Type, Config: spec.Config, Tasks: make(map[string]bool)}
	_, err = m.upsertDaemonSet(ctx, ds)
	if err != nil {
		return task.Respond(&daemonset.DaemonSetUpsertResponse{DaemonSetId: ds.DaemonSetId, State: daemonset.StateFailure, Error: err.Error()})
	}
	return task.Respond(&daemonset.DaemonSetUpsertResponse{DaemonSetId: ds.DaemonSetId, State: daemonset.StateSuccess})
}

// HandleTaskAssign handles runner tasks for assigning a new daemon task to a daemon set
func (m *Manager) HandleTaskAssign(ctx context.Context, req *task.Request) task.Response {
	spec := new(daemonset.DaemonTaskAssignRequest)
	err := json.Unmarshal(req.Task.Data, spec)
	if err != nil {
		return task.Error(err)
	}

	m.lock.Lock(spec.Type)
	defer m.lock.Unlock(spec.Type)

	// check if the daemon set is running
	ds, running := m.get(spec.Type)
	if !running {
		errMsg := fmt.Sprintf("no daemon set of type [%s] is currently running", spec.Type)
		return task.Respond(&daemonset.DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: daemonset.StateFailure, Error: errMsg})
	}

	// check if daemon set is already running a task with the given ID
	_, ok := ds.Tasks[spec.DaemonTaskId]
	if ok {
		errMsg := fmt.Sprintf("task with id [%s] is already running in daemon set of type [%s]", spec.DaemonTaskId, spec.Type)
		return task.Respond(&daemonset.DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: daemonset.StateFailure, Error: errMsg})
	}

	daemonTask := daemonset.DaemonTask{ID: spec.DaemonTaskId, Params: spec.Params}
	_, err = m.assignDaemonTasks(ctx, ds, &daemonset.DaemonTasks{Tasks: []daemonset.DaemonTask{daemonTask}})
	if err != nil {
		return task.Error(err)
	}
	return task.Respond(&daemonset.DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: daemonset.StateSuccess})
}

// upsertDaemonSet handles upserting a daemon set
func (m *Manager) upsertDaemonSet(ctx context.Context, ds *daemonset.DaemonSet) (*daemonset.DaemonSet, error) {
	if runningWithIdenticalConfig := m.handleRunningWithSameConfig(ds); runningWithIdenticalConfig != nil {
		return runningWithIdenticalConfig, nil
	}
	binpath, err := m.download(ctx, ds)
	if err != nil {
		return nil, err
	}
	if err = m.handleRunningWithDifferentConfig(ds); err != nil {
		return nil, err
	}
	ds, err = m.driver.StartDaemonSet(binpath, ds)
	if err != nil {
		return nil, err
	}
	m.daemonsets.Store(ds.Type, ds)
	utils.DsLogger(ds).Info("started daemon set process")
	return ds, nil
}

// assignDaemonTasks handles assigning daemon tasks to a daemon set
func (m *Manager) assignDaemonTasks(ctx context.Context, ds *daemonset.DaemonSet, tasks *daemonset.DaemonTasks) (*daemonset.DaemonSet, error) {
	taskIds := make([]string, len(tasks.Tasks))
	for i, s := range tasks.Tasks {
		taskIds[i] = s.ID
	}
	utils.DsLogger(ds).Infof("assigning tasks %s to daemon set", taskIds)
	_, err := m.driver.AssignDaemonTasks(ctx, ds, tasks)
	if err != nil {
		return nil, err
	}
	// insert the new task IDs in the daemonset's task set
	for _, taskId := range taskIds {
		ds.Tasks[taskId] = true
	}
	return ds, nil
}

// removeDaemonTasks handles removing daemon tasks from a daemon set
func (m *Manager) removeDaemonTasks(ctx context.Context, ds *daemonset.DaemonSet, taskIds *[]string) (*daemonset.DaemonSet, error) {
	utils.DsLogger(ds).Infof("removing tasks %s from daemon set", taskIds)
	_, err := m.driver.RemoveDaemonTasks(ctx, ds, taskIds)
	if err != nil {
		return nil, err
	}
	// insert the new task IDs in the daemonset's task set
	for _, taskId := range *taskIds {
		delete(ds.Tasks, taskId)
	}
	return ds, nil
}

// check if the daemon set is already running with same config as requested
// if this is the case, set the currently running daemon set's ID to the one passed in the request, and return true
// otherwise, return false
func (m *Manager) handleRunningWithSameConfig(ds *daemonset.DaemonSet) *daemonset.DaemonSet {
	dsOld, running := m.get(ds.Type)
	if running {
		// check if the config passed in the request is the same as the existing daemon set's
		if reflect.DeepEqual(dsOld.Config, ds.Config) {
			// If the configs are the same, no need to restart the daemon set
			utils.DsLogger(ds).Infof("daemon set of type [%s] is running with identical configuration. "+
				"Resetting its id to [%s]", ds.Type, ds.DaemonSetId)
			dsOld.DaemonSetId = ds.DaemonSetId
			return dsOld
		}
	}
	return nil
}

// check if the daemon set is already running with config different from requested
// if this is the case, kill the current daemon set process
func (m *Manager) handleRunningWithDifferentConfig(ds *daemonset.DaemonSet) error {
	dsOld, running := m.get(ds.Type)
	if running {
		utils.DsLogger(ds).Infof("daemon set of type [%s] is running. Killing it now", ds.Type)
		err := m.driver.StopDaemonSet(dsOld)
		if err != nil {
			utils.DsLogger(ds).WithError(err).Error("failed to kill daemon set process")
			return err
		}
		m.daemonsets.Delete(ds.Type)
	}
	return nil
}

// download the daemon set's executable file
func (m *Manager) download(ctx context.Context, ds *daemonset.DaemonSet) (string, error) {
	if ds.Config.ExecutableConfig == nil {
		return "", fmt.Errorf("no executable configuration provided for daemon set")
	}
	path, err := m.downloader.Download(ctx, ds.Type, ds.Config.Repository, ds.Config.ExecutableConfig)
	if err != nil {
		logrus.WithError(err).Error("task code download failed")
		return "", err
	}
	return path, nil
}

// get will return a *DaemonSet struct from the m.daemonsets synchronized map
// the returned struct (if present) will be type-asserted to the *DaemonSet type
func (m *Manager) get(t string) (*daemonset.DaemonSet, bool) {
	ds, ok := m.daemonsets.Load(t)
	if !ok {
		return nil, false
	}
	return ds.(*daemonset.DaemonSet), true
}
