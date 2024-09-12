// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package daemonset

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/drone/go-task/task/download"
	dsclient "github.com/harness/runner/delegateshell/daemonset/client"
	"github.com/harness/runner/delegateshell/daemonset/drivers"
	"github.com/sirupsen/logrus"
)

var (
	taskYmlPath = "task.yml"
)

type DaemonSetManager struct {
	// we use `sync.Map` to store running daemon sets. This is because, even though operations are atomic
	// per daemon set type (see comment on `lock` below), there can be multiple simultaneous operations
	// for different daemon set types. `sync.Map` is an appropriate date structure for this use case.
	// From https://pkg.go.dev/sync#Map: "the (sync.Map) type is optimized for two common use cases: ...
	// when multiple goroutines read, write, and overwrite entries for disjoint sets of keys. ..."
	daemonsets *sync.Map
	downloader download.Downloader
	driver     *drivers.HttpServerDriver
	// the `lock` here is a wrapper for a map of locks, indexed by daemon set's type
	// so that we can make sure operations are atomic for each daemon set type
	lock *KeyLock
}

func NewDaemonSetManager(d download.Downloader, isK8s bool) *DaemonSetManager {
	// TODO: Add suport for daemon sets in k8s runner
	return &DaemonSetManager{downloader: d, daemonsets: &sync.Map{}, lock: NewKeyLock(), driver: drivers.NewHttpServerDriver()}
}

// Get will return a *DaemonSet struct from the d.daemonsets synchronized map
// the returned struct (if present) will be type-asserted to the *DaemonSet type
func (d *DaemonSetManager) Get(t string) (*dsclient.DaemonSet, bool) {
	ds, ok := d.daemonsets.Load(t)
	if !ok {
		return nil, false
	}
	return ds.(*dsclient.DaemonSet), true
}

func (d *DaemonSetManager) GetAllTypes() map[string]bool {
	m := make(map[string]bool)
	d.daemonsets.Range(func(key, value interface{}) bool {
		ds := value.(*dsclient.DaemonSet)
		m[ds.Type] = true
		return true
	})
	return m
}

// UpsertDaemonSet is an idempotent method for upserting daemon sets
// returns the list of tasks assigned to the daemon set
func (d *DaemonSetManager) UpsertDaemonSet(ctx context.Context, dsId string, dsType string, dsConfig *dsclient.DaemonSetOperationalConfig) (*dsclient.DaemonTasks, error) {
	d.lock.Lock(dsType)
	defer d.lock.Unlock(dsType)

	// check if daemon set already exists in daemon set map
	ds, ok := d.Get(dsType)
	// if daemon set is running healthy with same config as requested,
	// set the currently running daemon set's ID to the one passed in the request,
	// and and return the tasks assigned to it
	if ok && d.isConfigIdentical(ds, dsConfig) {
		tasks, err := d.driver.ListDaemonTasks(ctx, ds)
		if err == nil {
			ds.DaemonSetId = dsId
			return tasks, nil
		}
		dsLogger(ds).Error("failed to list tasks, respawning this daemon set")
	}

	ds = &dsclient.DaemonSet{DaemonSetId: dsId, Type: dsType, Config: dsConfig}
	return d.startDaemonSet(ctx, ds)
}

// RemoveDaemonSet handles removing a daemon set
func (d *DaemonSetManager) RemoveDaemonSet(dsType string) {
	d.lock.Lock(dsType)
	defer d.lock.Unlock(dsType)

	ds, ok := d.Get(dsType)
	if ok {
		err := d.driver.StopDaemonSet(ds)
		if err != nil {
			dsLogger(ds).WithError(err).Warn("failed to kill daemon set process")
		}
		d.daemonsets.Delete(dsType)
	}
}

// SyncDaemonSet will check if the daemon set of given type is healthy and running,
// if the daemon set is not running, it will attempt to re-spawn it
func (d *DaemonSetManager) SyncDaemonSet(ctx context.Context, dsType string) {
	d.lock.Lock(dsType)
	defer d.lock.Unlock(dsType)

	// check if daemon set already exists in daemon set map
	ds, ok := d.Get(dsType)
	if !ok {
		// daemon set does not exist
		return
	}
	// check if daemon set is running healthy
	if !ds.Healthy {
		// daemon set has already been flagged as unhealthy
		// no point trying to restart it
		return
	}
	_, err := d.driver.ListDaemonTasks(ctx, ds)
	if err == nil {
		// daemon set is healthy and running
		return
	}
	dsLogger(ds).Error("failed to list tasks, respawning this daemon set")
	d.startDaemonSet(ctx, ds)
}

// ListDaemonTasks handles listing the tasks assigned to a daemon set
func (d *DaemonSetManager) ListDaemonTasks(ctx context.Context, dsType string) (*dsclient.DaemonTasks, error) {
	ds, ok := d.Get(dsType)
	if !ok {
		return nil, fmt.Errorf("daemon set of type [%s] does not exist", dsType)
	}

	tasks, err := d.driver.ListDaemonTasks(ctx, ds)
	if err != nil {
		return nil, err
	}

	return tasks, nil

}

// AssignDaemonTasks handles assigning daemon tasks to a daemon set
func (d *DaemonSetManager) AssignDaemonTasks(ctx context.Context, dsType string, tasks *dsclient.DaemonTasks) error {
	ds, ok := d.Get(dsType)
	if !ok {
		return fmt.Errorf("daemon set of type [%s] does not exist", dsType)
	}

	taskIds := make([]string, len(tasks.Tasks))
	for i, s := range tasks.Tasks {
		taskIds[i] = s.ID
	}

	dsLogger(ds).Infof("assigning tasks %s to daemon set", taskIds)
	_, err := d.driver.AssignDaemonTasks(ctx, ds, tasks)
	if err != nil {
		dsLogger(ds).Errorf("failed to assign tasks %s to daemon set", taskIds)
		return err
	}

	return nil
}

// RemoveDaemonTasks handles removing daemon tasks from a daemon set
func (d *DaemonSetManager) RemoveDaemonTasks(ctx context.Context, dsType string, taskIds *[]string) error {
	ds, ok := d.Get(dsType)
	if !ok {
		return fmt.Errorf("daemon set of type [%s] does not exist", dsType)
	}

	dsLogger(ds).Infof("removing tasks %s from daemon set", *taskIds)
	_, err := d.driver.RemoveDaemonTasks(ctx, ds, taskIds)
	if err != nil {
		dsLogger(ds).WithError(err).Errorf("failed to remove tasks %s from daemon set", *taskIds)
		return err
	}

	return nil
}

// startDaemonSet is an internal method which is used to start daemon sets
// calling this method should always be wrapped by a lock in the daemon set's type,
// since here we are both reading and writing to the `d.daemonsets` map
func (d *DaemonSetManager) startDaemonSet(ctx context.Context, ds *dsclient.DaemonSet) (*dsclient.DaemonTasks, error) {
	tasks := &dsclient.DaemonTasks{}

	binpath, err := d.download(ctx, ds)
	if err != nil {
		ds.Healthy = false
		d.daemonsets.Store(ds.Type, ds)
		return tasks, err
	}

	// if daemon set exists in daemon set map
	// attempt to kill its process and remove
	// it from the map
	_, ok := d.Get(ds.Type)
	if ok {
		err := d.driver.StopDaemonSet(ds)
		if err != nil {
			dsLogger(ds).WithError(err).Warn("failed to kill daemon set process")
		}
		d.daemonsets.Delete(ds.Type)
	}

	serverInfo, err := d.driver.StartDaemonSet(binpath, ds)
	if err != nil {
		ds.Healthy = false
		d.daemonsets.Store(ds.Type, ds)
		return tasks, err
	}
	ds.ServerInfo = serverInfo

	tasks, err = d.waitForHealthyState(ctx, ds)
	if err != nil {
		ds.Healthy = false
		d.daemonsets.Store(ds.Type, ds)
		return tasks, err
	}

	ds.Healthy = true
	d.daemonsets.Store(ds.Type, ds)
	dsLogger(ds).Info("started daemon set process")
	return tasks, nil
}

// isConfigIdentical checks whether a given daemon set's config is identical to `dsConfig`
func (d *DaemonSetManager) isConfigIdentical(ds *dsclient.DaemonSet, dsConfig *dsclient.DaemonSetOperationalConfig) bool {
	if reflect.DeepEqual(ds.Config, dsConfig) {
		return true
	}
	return false
}

// download the daemon set's executable file
func (d *DaemonSetManager) download(ctx context.Context, ds *dsclient.DaemonSet) (string, error) {
	if ds.Config.ExecutableConfig == nil {
		return "", fmt.Errorf("no executable configuration provided for daemon set")
	}
	path, err := d.downloader.Download(ctx, ds.Type, ds.Config.Repository, ds.Config.ExecutableConfig)
	if err != nil {
		logrus.WithError(err).Error("task code download failed")
		return "", err
	}
	return path, nil
}

// waitForHealthyState pools a daemon set tasks list API until successful and returns the list of tasks
func (d *DaemonSetManager) waitForHealthyState(ctx context.Context, ds *dsclient.DaemonSet) (*dsclient.DaemonTasks, error) {
	timeout := 3 * time.Minute
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		// Every 5 seconds, try to call ListDaemonTasks
		case <-ticker.C:
			tasks, err := d.driver.ListDaemonTasks(ctx, ds)
			if err == nil {
				return tasks, nil
			}

		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("health check timeout reached: failed to list tasks after %f minutes", timeout.Minutes())
		}
	}
}

// returns a logrus *Entry with daemon set's data as fields
func dsLogger(ds *dsclient.DaemonSet) *logrus.Entry {
	logger := logrus.WithField("id", ds.DaemonSetId).
		WithField("type", ds.Type)
	if ds.ServerInfo != nil {
		logger = logger.WithField("port", ds.ServerInfo.Port).
			WithField("pid", ds.ServerInfo.Execution.Process.Pid).
			WithField("binpath", ds.ServerInfo.Execution.Path)
	}
	return logger
}
