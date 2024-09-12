// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package daemonset

import (
	"context"
	"time"

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
	stopChannel      chan struct{}
}

func NewDaemonSetReconciler(daemonSetManager *DaemonSetManager, managerClient *client.ManagerClient) *DaemonSetReconciler {
	// TODO: Add suport for daemon sets in k8s runner
	return &DaemonSetReconciler{daemonSetManager: daemonSetManager, managerClient: managerClient, stopChannel: make(chan struct{})}
}

func (d *DaemonSetReconciler) Start(ctx context.Context, id string, interval time.Duration) error {
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
			case <-d.stopChannel:
				logrus.Infoln("stopped daemon set reconciliation job")
				return
			case <-timer.C:
				taskEventsCtx, cancelFn := context.WithTimeout(ctx, reconcileTimeout)
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

func (d *DaemonSetReconciler) Stop() {
	close(d.stopChannel) // Notify to stop reconcile job
}

func (d *DaemonSetReconciler) reconcile(ctx context.Context, runnerId string) error {
	daemonSetTypes := d.daemonSetManager.GetAllTypes()
	d.syncDaemonSets(ctx, daemonSetTypes)
	req := d.getReconcileRequest(daemonSetTypes)
	resp, err := d.managerClient.ReconcileDaemonSets(ctx, runnerId, &req)
	if err != nil {
		return err
	}
	d.syncDaemonSetsWithHarnessServer(ctx, runnerId, daemonSetTypes, resp)
	return nil
}

func (d *DaemonSetReconciler) syncDaemonSets(ctx context.Context, dsTypes map[string]bool) {
	for dsType := range dsTypes {
		d.daemonSetManager.SyncDaemonSet(ctx, dsType)
	}
}

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

func (d *DaemonSetReconciler) syncDaemonSetsWithHarnessServer(ctx context.Context, runnerId string, dsTypesFromRunner map[string]bool, resp *client.DaemonSetReconcileResponse) {
	// kill daemon sets which are not supposed to be running
	dsTypesFromServer := GetAllDaemonSetTypes(resp)
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
			d.acquireAndAssignDaemonTasks(ctx, runnerId, dsId, dsType, taskIdsToAssign)
		}
	}
}

func (d *DaemonSetReconciler) acquireAndAssignDaemonTasks(ctx context.Context, runnerId string, dsId string, dsType string, taskIds []string) {
	resp, err := d.managerClient.AcquireDaemonTasks(ctx, runnerId, dsId, &client.DaemonTaskAcquireRequest{TaskIds: taskIds})
	if err != nil {
		logrus.WithError(err).Errorf("failed to acquire daemon task params during reconcile: id [%s]; type [%s]", dsId, dsType)
	}
	var daemonTasks []dsclient.DaemonTask
	for _, taskAssignRequest := range resp.Tasks {
		daemonTasks = append(daemonTasks, dsclient.DaemonTask{ID: taskAssignRequest.DaemonTaskId, Params: taskAssignRequest.Params})
	}
	_ = d.daemonSetManager.AssignDaemonTasks(ctx, dsType, &dsclient.DaemonTasks{Tasks: daemonTasks})
}

func GetAllDaemonSetTypes(resp *client.DaemonSetReconcileResponse) map[string]bool {
	set := make(map[string]bool)
	if resp.Data != nil {
		for _, entry := range resp.Data {
			set[entry.Type] = true
		}
	}
	return set
}

func compareTasks(serverResponse client.DaemonSetReconcileResponseEntry, runnerResponse *dsclient.DaemonTasks) ([]string, []string) {
	taskIdsFromRunner := make(map[string]bool)
	for _, task := range runnerResponse.Tasks {
		taskIdsFromRunner[task.ID] = true
	}
	return compareSets(listToSet(serverResponse.TaskIds), taskIdsFromRunner)
}

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

func listToSet(l []string) map[string]bool {
	set := make(map[string]bool)
	for _, v := range l {
		set[v] = true
	}
	return set
}
