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
	"time"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/download"
	"github.com/harness/runner/daemonset"
	"github.com/harness/runner/daemonset/drivers"
	"github.com/harness/runner/daemonset/utils"
	"github.com/harness/runner/delegateshell/client"
	"github.com/sirupsen/logrus"
)

var (
	taskYmlPath      = "task.yml"
	reconcileTimeout = 300 * time.Second
)

type Manager struct {
	// we use `sync.Map` to store running daemon sets. This is because, even though operations are atomic
	// per daemon set type (see comment on `lock` below), there can be multiple simultaneous operations
	// for different daemon set types. `sync.Map` is an appropriate date structure for this use case.
	// From https://pkg.go.dev/sync#Map: "the (sync.Map) type is optimized for two common use cases: ...
	// when multiple goroutines read, write, and overwrite entries for disjoint sets of keys. ..."
	managerClient *client.ManagerClient
	daemonsets    *sync.Map
	downloader    download.Downloader
	driver        *drivers.HttpServerDriver
	// the `lock` here is a wrapper for a map of locks, indexed by daemon set's type
	// so that we can make sure operations are atomic for each daemon set type
	lock        *KeyLock
	stopChannel chan struct{}
}

// New returns the daemon set task execution driver
func New(managerClient *client.ManagerClient, d download.Downloader, isK8s bool) *Manager {
	// TODO: Add suport for daemon sets in k8s runner
	return &Manager{managerClient: managerClient, downloader: d, daemonsets: &sync.Map{}, lock: NewKeyLock(), driver: drivers.NewHttpServerDriver(), stopChannel: make(chan struct{})}
}

// PollRunnerEvents continually asks the task server for tasks to execute.
func (m *Manager) StartReconcile(ctx context.Context, id string, interval time.Duration) error {
	// Task event poller
	go func() {
		timer := time.NewTimer(interval)
		defer timer.Stop()
		for {
			timer.Reset(interval)
			select {
			case <-ctx.Done():
				logrus.Errorln("context canceled during reconcile flow, this should not happen")
				return
			case <-m.stopChannel:
				logrus.Infoln("stopped daemon set reconciliation job")
				return
			case <-timer.C:
				taskEventsCtx, cancelFn := context.WithTimeout(ctx, reconcileTimeout)
				err := m.reconcile(taskEventsCtx, id)
				if err != nil {
					logrus.WithError(err).Errorf("daemon set reconciliation failed")
				}
				cancelFn()
			}
		}
	}()
	logrus.Infof("initialized reconcile flow for daemon sets!")
	return nil
}

func (m *Manager) StopReconcile() {
	close(m.stopChannel) // Notify to stop reconcile job
}

func (m *Manager) reconcile(ctx context.Context, runnerId string) error {
	// lock all daemon set operations
	logrus.Info("Starting reconcile flow...")
	m.lock.LockAll()
	defer m.lock.UnlockAll()

	m.syncAllWithDaemonSetServers(ctx)
	req := m.getReconcileRequest()
	resp, err := m.managerClient.ReconcileDaemonSets(ctx, runnerId, &req)
	if err != nil {
		return err
	}
	m.syncAllWithHarnessServer(ctx, runnerId, resp)
	logrus.Info("Done with reconcile flow...")
	return nil
}

func (m *Manager) syncAllWithHarnessServer(ctx context.Context, id string, resp *client.DaemonSetReconcileResponse) {
	// kill daemon sets which are not supposed to be running
	dsTypesFromServer := utils.GetAllDaemonSetTypes(resp)
	dsTypesFromRunner := utils.GetAllKeysFromSyncMap(m.daemonsets)
	dsTypesToStop, _ := utils.CompareSets(dsTypesFromServer, dsTypesFromRunner)
	for _, dsType := range dsTypesToStop {
		m.stopDaemonSetIfRunning(dsType)
	}

	// ensure all daemon sets reported by server are running
	for _, e := range resp.Data {
		var ds *daemonset.DaemonSet
		var err error
		oldDs := m.getDaemonSetRunningWithIdenticalConfig(e.DaemonSetId, e.Type, e.Config)
		if oldDs == nil {
			ds, err = m.upsertDaemonSet(ctx, e.DaemonSetId, e.Type, e.Config)
			if err != nil {
				continue
			}
		} else {
			ds = oldDs
		}
		tasksFromServer := utils.ListToSet(e.TaskIds)
		tasksToRemove, tasksToAssign := utils.CompareSets(tasksFromServer, ds.Tasks)
		if len(tasksToRemove) > 0 {
			_ = m.removeDaemonTasks(ctx, ds, &tasksToRemove)
		}
		if len(tasksToAssign) > 0 {
			resp, err := m.managerClient.AcquireDaemonTasks(ctx, id, ds.DaemonSetId, &client.DaemonTaskAcquireRequest{TaskIds: tasksToAssign})
			if err != nil {
				utils.DsLogger(ds).WithError(err).Error("failed to acquire task params during reconcile")
				continue
			}
			var daemonTasks []daemonset.DaemonTask
			for _, taskAssignRequest := range resp.Tasks {
				daemonTasks = append(daemonTasks, daemonset.DaemonTask{ID: taskAssignRequest.DaemonTaskId, Params: taskAssignRequest.Params})
			}
			_ = m.assignDaemonTasks(ctx, ds, &daemonset.DaemonTasks{Tasks: daemonTasks})
		}
		utils.DsLogger(ds).Info("Done doing reconciliation for this daemon set")
	}
}

func (m *Manager) getReconcileRequest() client.DaemonSetReconcileRequest {
	data := []client.DaemonSetReconcileRequestEntry{}
	m.daemonsets.Range(func(key, value interface{}) bool {
		ds := value.(*daemonset.DaemonSet)
		e := client.DaemonSetReconcileRequestEntry{DaemonSetId: ds.DaemonSetId, Type: ds.Type, Config: ds.Config, Healthy: ds.Healthy}
		data = append(data, e)
		return true
	})
	return client.DaemonSetReconcileRequest{Data: data}
}

func (m *Manager) syncAllWithDaemonSetServers(ctx context.Context) {
	// Iterate over m.daemonsets and call the task list API for each daemon set
	m.daemonsets.Range(func(key, value interface{}) bool {
		ds := value.(*daemonset.DaemonSet)
		m.syncWithDaemonSetServer(ctx, ds)
		return true
	})
}

func (m *Manager) syncWithDaemonSetServer(ctx context.Context, ds *daemonset.DaemonSet) {
	if !ds.Healthy {
		return
	}
	tasks, err := m.driver.FetchDaemonTasks(ctx, ds)
	if err != nil {
		utils.DsLogger(ds).Warn("failed to list daemon tasks, re-spawning this daemon set")
		ds, err := m.upsertDaemonSet(ctx, ds.DaemonSetId, ds.Type, ds.Config)
		if err != nil {
			utils.DsLogger(ds).WithError(err).Error("failed to re-spawn daemon set")
		}
		return
	}
	utils.DsLogger(ds).Infof("Got the following tasks from daemon set %+v", *tasks)
	tasksMap := make(map[string]bool)
	for _, daemonTask := range tasks.Tasks {
		tasksMap[daemonTask.ID] = true
	}
	ds.Tasks = tasksMap
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

	ds := m.getDaemonSetRunningWithIdenticalConfig(spec.DaemonSetId, spec.Type, spec.Config)
	if ds != nil {
		return task.Respond(&daemonset.DaemonSetUpsertResponse{DaemonSetId: ds.DaemonSetId, State: daemonset.StateSuccess})
	}

	ds, err = m.upsertDaemonSet(ctx, spec.DaemonSetId, spec.Type, spec.Config)
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
	err = m.assignDaemonTasks(ctx, ds, &daemonset.DaemonTasks{Tasks: []daemonset.DaemonTask{daemonTask}})
	if err != nil {
		return task.Respond(&daemonset.DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: daemonset.StateFailure, Error: err.Error()})
	}
	return task.Respond(&daemonset.DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: daemonset.StateSuccess})
}

// upsertDaemonSet handles upserting a daemon set
func (m *Manager) upsertDaemonSet(ctx context.Context, dsId string, dsType string, dsConfig daemonset.DaemonSetOperationalConfig) (*daemonset.DaemonSet, error) {
	ds := &daemonset.DaemonSet{DaemonSetId: dsId, Type: dsType, Config: dsConfig, Tasks: make(map[string]bool)}

	binpath, err := m.download(ctx, ds)
	if err != nil {
		ds.Healthy = false
		m.daemonsets.Store(ds.Type, ds)
		return ds, err
	}
	m.stopDaemonSetIfRunning(dsType)
	serverInfo, err := m.driver.StartDaemonSet(binpath, ds)
	if err != nil {
		ds.Healthy = false
		m.daemonsets.Store(ds.Type, ds)
		return ds, err
	}
	ds.ServerInfo = serverInfo
	ds.Healthy = true
	m.daemonsets.Store(ds.Type, ds)
	utils.DsLogger(ds).Info("started daemon set process")
	return ds, nil
}

// assignDaemonTasks handles assigning daemon tasks to a daemon set
func (m *Manager) assignDaemonTasks(ctx context.Context, ds *daemonset.DaemonSet, tasks *daemonset.DaemonTasks) error {
	taskIds := make([]string, len(tasks.Tasks))
	for i, s := range tasks.Tasks {
		taskIds[i] = s.ID
	}
	utils.DsLogger(ds).Infof("assigning tasks %s to daemon set", taskIds)
	_, err := m.driver.AssignDaemonTasks(ctx, ds, tasks)
	if err != nil {
		utils.DsLogger(ds).Errorf("failed to assign tasks %s to daemon set", taskIds)
		return err
	}
	// insert the new task IDs in the daemonset's task set
	for _, taskId := range taskIds {
		ds.Tasks[taskId] = true
	}
	return nil
}

// removeDaemonTasks handles removing daemon tasks from a daemon set
func (m *Manager) removeDaemonTasks(ctx context.Context, ds *daemonset.DaemonSet, taskIds *[]string) error {
	utils.DsLogger(ds).Infof("removing tasks %s from daemon set", *taskIds)
	_, err := m.driver.RemoveDaemonTasks(ctx, ds, taskIds)
	if err != nil {
		utils.DsLogger(ds).WithError(err).Errorf("failed to remove tasks %s from daemon set", *taskIds)
		return err
	}
	// insert the new task IDs in the daemonset's task set
	for _, taskId := range *taskIds {
		delete(ds.Tasks, taskId)
	}
	return nil
}

// check if the daemon set is already running with same config as requested
// if this is the case, set the currently running daemon set's ID to the one passed in the request, and return true
// otherwise, return false
func (m *Manager) getDaemonSetRunningWithIdenticalConfig(dsId string, dsType string, dsConfig daemonset.DaemonSetOperationalConfig) *daemonset.DaemonSet {
	// TODO: Here we need to check whether the pre-existing daemon set is healthy, before returning it.
	// This will be implemented in the next PR.
	dsOld, running := m.get(dsType)
	if running {
		// check if the config passed in the request is the same as the existing daemon set's
		if reflect.DeepEqual(dsOld.Config, dsConfig) {
			// If the configs are the same, no need to restart the daemon set
			// TODO: maybe we should check if daemon set is healthy here too
			if dsOld.DaemonSetId == dsId {
				return dsOld
			}
			utils.DsLogger(dsOld).Infof("daemon set of type [%s] is running with identical configuration. "+
				"Resetting its id to [%s]", dsType, dsId)
			dsOld.DaemonSetId = dsId
			return dsOld
		}
	}
	return nil
}

// check if the daemon set is already running with config different from requested
// if this is the case, attempt to kill the current daemon set process
func (m *Manager) stopDaemonSetIfRunning(dsType string) {
	ds, running := m.get(dsType)
	if running {
		utils.DsLogger(ds).Infof("daemon set of type [%s] is running. Killing it now", dsType)
		err := m.driver.StopDaemonSet(ds)
		if err != nil {
			utils.DsLogger(ds).WithError(err).Warn("failed to kill daemon set process")
		}
		m.daemonsets.Delete(dsType)
	}
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
