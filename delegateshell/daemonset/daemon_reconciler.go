// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package daemonset

import (
	"context"
	"encoding/json"
	"time"

	"github.com/drone/go-task/task"
	"github.com/harness/runner/delegateshell/client"
	dsclient "github.com/harness/runner/delegateshell/daemonset/client"
	"github.com/sirupsen/logrus"
)

var (
	reconcileTimeout = 10 * time.Minute
)

type DaemonSetReconciler struct {
	daemonSetManager *DaemonSetManager
	managerClient    *client.ManagerClient
	router           *task.Router // used for resolving secrets in daemon tasks
	ctx              context.Context
	cancelCtx        context.CancelFunc
	doneChannel      chan bool
}

func NewDaemonSetReconciler(ctx context.Context, daemonSetManager *DaemonSetManager, router *task.Router, managerClient *client.ManagerClient) *DaemonSetReconciler {
	ctx, cancelCtx := context.WithCancel(ctx)
	return &DaemonSetReconciler{
		daemonSetManager: daemonSetManager,
		managerClient:    managerClient,
		router:           router,
		ctx:              ctx,
		cancelCtx:        cancelCtx,
		doneChannel:      make(chan bool),
	}
}

// Start will start the daemon set reconciling job
func (d *DaemonSetReconciler) Start(id string, interval time.Duration) error {
	// Task event poller
	go func() {
		timer := time.NewTimer(interval)
		defer timer.Stop()
		for {
			timer.Reset(interval)
			select {
			case <-d.ctx.Done():
				close(d.doneChannel)
				logrus.Info("stopped daemon set reconciliation job")
				return
			case <-timer.C:
				taskEventsCtx, cancelFn := context.WithTimeout(d.ctx, reconcileTimeout)
				err := d.reconcile(taskEventsCtx, id)
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

// Stop will stop the daemon set reconciling job
func (d *DaemonSetReconciler) Stop() {
	logrus.Info("cancelling daemon set reconciliation job")
	d.cancelCtx()
	<-d.doneChannel // block until last reconciliation returns
}

// reconcile runs the daemon set reconcile flow
func (d *DaemonSetReconciler) reconcile(ctx context.Context, runnerId string) error {
	daemonSetTypes := d.daemonSetManager.GetAllTypes()
	d.syncDaemonSets(ctx, daemonSetTypes)
	req := d.getReconcileRequest(daemonSetTypes)
	resp, err := d.managerClient.ReconcileDaemonSets(ctx, runnerId, &req)
	if err != nil {
		return err
	}
	d.syncWithHarnessServer(ctx, runnerId, daemonSetTypes, resp)
	return nil
}

// syncDaemonSets will call DaemonSetManager.SyncDaemonSet,
// for the set of daemon set types passed as arguments
func (d *DaemonSetReconciler) syncDaemonSets(ctx context.Context, dsTypes map[string]bool) {
	for dsType := range dsTypes {
		d.daemonSetManager.SyncDaemonSet(ctx, dsType)
	}
}

// getReconcileRequest generates the reconcile request that
// will be sent to Harness manager
func (d *DaemonSetReconciler) getReconcileRequest(daemonSetTypes map[string]bool) client.DaemonSetReconcileRequest {
	data := []client.DaemonSetReconcileRequestEntry{}
	for dsType := range daemonSetTypes {
		ds, ok := d.daemonSetManager.Get(dsType)
		if !ok {
			continue
		}
		e := client.DaemonSetReconcileRequestEntry{DaemonSetId: ds.DaemonSetId, Type: ds.Type, Config: *ds.Config, Healthy: ds.Healthy}
		data = append(data, e)
	}
	return client.DaemonSetReconcileRequest{Data: data}
}

// syncWithHarnessServer attempts to ensure the state of the daemon sets is as reported by the server
// i.e. the daemon sets reported by the server are running; and
// the tasks for each daemon set are assigned, as reported by the server
func (d *DaemonSetReconciler) syncWithHarnessServer(ctx context.Context, runnerId string, dsTypesFromRunner map[string]bool, resp *client.DaemonSetReconcileResponse) {
	// kill daemon sets which are not supposed to be running
	dsTypesFromServer := getAllDaemonSetTypes(resp)
	dsTypesToRemove, _ := compareSets(dsTypesFromServer, dsTypesFromRunner)
	for _, dsType := range dsTypesToRemove {
		d.daemonSetManager.RemoveDaemonSet(dsType)
	}

	// ensure all daemon sets reported by server are running
	for _, e := range resp.Data {
		if e.SkipUpdate {
			continue
		}
		dsId := e.DaemonSetId
		dsType := e.Type
		tasksFromRunner, err := d.daemonSetManager.UpsertDaemonSet(ctx, dsId, dsType, &e.Config)
		if err != nil {
			logrus.WithError(err).Errorf("failed sync daemon set with server: id [%s]; type [%s]", dsId, dsType)
			continue
		}
		taskIdsToRemove, taskIdsToAssign := compareTasks(e, tasksFromRunner)
		if len(taskIdsToRemove) > 0 {
			_ = d.daemonSetManager.RemoveDaemonTasks(ctx, dsType, &taskIdsToRemove)
		}
		if len(taskIdsToAssign) > 0 {
			d.acquireAndAssignDaemonTasks(ctx, runnerId, dsId, dsType, &taskIdsToAssign)
		}
	}
}

// acquireAndAssignDaemonTasks fetches params of tasks from Harness manager
// and assigns these tasks to a daemon set of the given type (`dsType`)
func (d *DaemonSetReconciler) acquireAndAssignDaemonTasks(ctx context.Context, runnerId string, dsId string, dsType string, taskIds *[]string) {
	resp, err := d.managerClient.AcquireDaemonTasks(ctx, runnerId, &client.DaemonTaskAcquireRequest{TaskIds: *taskIds})
	if err != nil {
		logrus.WithError(err).Errorf("failed to acquire daemon task params during reconcile: id [%s]; type [%s]", dsId, dsType)
	}

	var daemonTasks []dsclient.DaemonTask
	for _, req := range resp.Requests {
		taskAssignRequest := new(client.DaemonTaskAssignRequest)
		err := json.Unmarshal(req.Task.Data, taskAssignRequest)
		if err != nil {
			logrus.WithError(err).Errorf("failed parsing data for request [%s], skipping this request", req.ID)
			continue
		}
		logrus.Infof("resolving secrets for daemon task [%s]", taskAssignRequest.DaemonTaskId)
		secrets, err := d.router.ResolveSecrets(ctx, req.Tasks)
		if err != nil {
			logrus.WithError(err).Errorf("failed to resolve secrets for task [%s], skipping this task", taskAssignRequest.DaemonTaskId)
			continue
		}
		daemonTasks = append(daemonTasks, dsclient.DaemonTask{ID: taskAssignRequest.DaemonTaskId, Params: taskAssignRequest.Params, Secrets: secrets})
	}

	_ = d.daemonSetManager.AssignDaemonTasks(ctx, dsType, &dsclient.DaemonTasks{Tasks: daemonTasks})
}

// getAllDaemonSetTypes returns a set of daemon set types that exist
// in the reconcile response from Harness manager
func getAllDaemonSetTypes(resp *client.DaemonSetReconcileResponse) map[string]bool {
	set := make(map[string]bool)
	if resp.Data != nil {
		for _, entry := range resp.Data {
			set[entry.Type] = true
		}
	}
	return set
}

// compareTasks compares a daemon set task data from server (Harness manager) with local data (daemon sets)
// it returns a pair of lists. The first one contains the taskIds that are in local data but not in server.
// the second one contains the taskIds that are in server but not in local data
func compareTasks(serverResponse client.DaemonSetServerInfo, runnerResponse *dsclient.DaemonTasksMetadata) ([]string, []string) {
	taskIdsFromRunner := make(map[string]bool)
	for _, task := range *runnerResponse {
		taskIdsFromRunner[task.ID] = true
	}
	return compareSets(listToSet(serverResponse.TaskIds), taskIdsFromRunner)
}

// compareSets compares two sets and returns entries that are
// missing in the first set (`missingInSet1`),
// and missing in the second set (`missingInSet2`)
func compareSets(set1, set2 map[string]bool) (missingInSet1, missingInSet2 []string) {
	// find items in set2 that are missing in set1
	for item := range set2 {
		if !set1[item] {
			missingInSet1 = append(missingInSet1, item)
		}
	}
	// find items in set1 that are missing in set2
	for item := range set1 {
		if !set2[item] {
			missingInSet2 = append(missingInSet2, item)
		}
	}
	return missingInSet1, missingInSet2
}

// listToSet converts a list to a set
func listToSet(l []string) map[string]bool {
	set := make(map[string]bool)
	for _, v := range l {
		set[v] = true
	}
	return set
}
