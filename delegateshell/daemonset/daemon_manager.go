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

	"github.com/drone/go-task/task/downloader"
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
	downloader downloader.Downloader
	driver     drivers.DaemonSetDriver
	// the `lock` here is a wrapper for a map of locks, indexed by daemon set's type
	// so that we can make sure operations are atomic for each daemon set type
	lock *KeyLock
}

func NewDaemonSetManager(d downloader.Downloader, isK8s bool) *DaemonSetManager {
	// TODO: Add suport for daemon sets in k8s runner. For this, we need to implement the `K8sServerDriver`.
	return &DaemonSetManager{downloader: d, daemonsets: &sync.Map{}, lock: NewKeyLock(), driver: drivers.NewLocalDriver()}
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

// GetAllTypes returns a set of all the daemon set types currently existing in `d.daemonsets`
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
func (d *DaemonSetManager) UpsertDaemonSet(ctx context.Context, dsId string, dsType string, dsConfig *dsclient.DaemonSetOperationalConfig) (*dsclient.DaemonTasksMetadata, error) {
	d.lock.Lock(dsType)
	defer d.lock.Unlock(dsType)

	// check if daemon set already exists in daemon set map
	ds, ok := d.Get(dsType)
	// if daemon set is running healthy with same config as requested,
	// set the currently running daemon set's ID to the one passed in the request,
	// and return the tasks assigned to it
	if ok && isConfigIdentical(ds.Config, dsConfig) {
		tasks, err := d.driver.ListDaemonTasks(ctx, ds)
		if err == nil {
			ds.DaemonSetId = dsId
			return tasks, nil
		}
		dsLogger(ds).Error("failed to list tasks, respawning this daemon set")
	}

	ds = &dsclient.DaemonSet{DaemonSetId: dsId, Type: dsType, Config: dsConfig}

	tasks, err := d.startDaemonSet(ctx, ds)
	if err == nil {
		ds.Healthy = true
	} else {
		ds.Healthy = false
	}
	d.daemonsets.Store(ds.Type, ds)
	return tasks, err
}

// SyncDaemonSet will check if the daemon set of given type is healthy and running,
// if the daemon set is not running, it will attempt to re-spawn it
// if the daemon set is not healthy, even after re-spawn, mark it as unhealthy
func (d *DaemonSetManager) SyncDaemonSet(ctx context.Context, dsType string) {
	d.lock.Lock(dsType)
	defer d.lock.Unlock(dsType)

	// check if daemon set already exists in daemon set map
	ds, ok := d.Get(dsType)
	if !ok {
		// daemon set does not exist
		logrus.Errorf("daemon set of type [%s] does not exist, skipping sync for it", dsType)
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

	_, err = d.startDaemonSet(ctx, ds)
	if err == nil {
		ds.Healthy = true
	} else {
		ds.Healthy = false
	}
	d.daemonsets.Store(ds.Type, ds)
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

// RemoveAllDaemonSets handles removing all daemon sets
func (d *DaemonSetManager) RemoveAllDaemonSets() {
	d.lock.LockAll()
	defer d.lock.UnlockAll()

	for dsType := range d.GetAllTypes() {
		ds, ok := d.Get(dsType)
		if ok {
			err := d.driver.StopDaemonSet(ds)
			if err != nil {
				dsLogger(ds).WithError(err).Warn("failed to kill daemon set process")
			}
			d.daemonsets.Delete(dsType)
		}
	}
}

// ListDaemonTasks handles listing the tasks assigned to a daemon set
func (d *DaemonSetManager) ListDaemonTasks(ctx context.Context, dsType string) (*dsclient.DaemonTasksMetadata, error) {
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
	dsLogger(ds).Infof("assigned tasks %s to daemon set", taskIds)

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
	dsLogger(ds).Infof("removed tasks %s from daemon set", *taskIds)

	return nil
}

// startDaemonSet is an internal method which is used to start daemon sets
// calling this method should always be wrapped by a lock in the daemon set's type
func (d *DaemonSetManager) startDaemonSet(ctx context.Context, ds *dsclient.DaemonSet) (*dsclient.DaemonTasksMetadata, error) {
	tasks := &dsclient.DaemonTasksMetadata{}

	binpath, err := d.download(ctx, ds)
	if err != nil {
		return tasks, err
	}

	// if daemon set exists in daemon set map
	// attempt to kill its process and remove
	// it from the map
	oldDs, ok := d.Get(ds.Type)
	if ok {
		err := d.driver.StopDaemonSet(oldDs)
		if err != nil {
			dsLogger(oldDs).WithError(err).Warn("failed to kill daemon set process")
		}
		d.daemonsets.Delete(oldDs.Type)
	}

	serverInfo, err := d.driver.StartDaemonSet(binpath, ds)
	if err != nil {
		return tasks, err
	}
	ds.ServerInfo = serverInfo

	tasks, err = d.waitForHealthyState(ctx, ds)
	if err != nil {
		return tasks, err
	}

	dsLogger(ds).Info("started daemon set process")
	return tasks, nil
}

// download the daemon set's executable file
func (d *DaemonSetManager) download(ctx context.Context, ds *dsclient.DaemonSet) (string, error) {
	if ds.Config.ExecutableConfig == nil {
		return "", fmt.Errorf("no executable configuration provided for daemon set")
	}
	// set daemon set's version in ExecutableConfig
	path, err := d.downloader.DownloadExecutable(ctx, ds.Type, ds.Config.Version, ds.Config.ExecutableConfig)
	if err != nil {
		logrus.WithError(err).Error("failed to download task executable file")
		return "", err
	}
	return path, nil
}

// waitForHealthyState pools a daemon set tasks list API until successful and returns the list of tasks
func (d *DaemonSetManager) waitForHealthyState(ctx context.Context, ds *dsclient.DaemonSet) (*dsclient.DaemonTasksMetadata, error) {
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

// isConfigIdentical checks whether two daemon set configs are the same
func isConfigIdentical(config1 *dsclient.DaemonSetOperationalConfig, config2 *dsclient.DaemonSetOperationalConfig) bool {
	if config1 == config2 {
		return true
	}
	if config1 == nil || config2 == nil {
		return false
	}
	if config1.Cpu != config2.Cpu {
		return false
	}
	if !reflect.DeepEqual(config1.Envs, config2.Envs) {
		return false
	}
	if config1.ExecutableConfig == nil || config2.ExecutableConfig == nil {
		return false
	}
	if !reflect.DeepEqual(config1.ExecutableConfig.Executables, config2.ExecutableConfig.Executables) {
		return false
	}
	if config1.MemoryBytes != config2.MemoryBytes {
		return false
	}
	if config1.Image != config2.Image {
		return false
	}
	if config1.Version != config2.Version {
		return false
	}
	return true
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
