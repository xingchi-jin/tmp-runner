// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package utils

import (
	"sync"

	"github.com/harness/runner/daemonset"
	"github.com/harness/runner/delegateshell/client"
	"github.com/sirupsen/logrus"
)

// returns a logrus *Entry with daemon set's data as fields
func DsLogger(ds *daemonset.DaemonSet) *logrus.Entry {
	logger := logrus.WithField("id", ds.DaemonSetId).
		WithField("type", ds.Type)
	if ds.ServerInfo != nil {
		logger = logger.WithField("port", ds.ServerInfo.Port).
			WithField("pid", ds.ServerInfo.Execution.Process.Pid).
			WithField("binpath", ds.ServerInfo.Execution.Path)
	}
	return logger
}

func ListToSet(l []string) map[string]bool {
	set := make(map[string]bool)
	for _, v := range l {
		set[v] = true
	}
	return set
}

func SetToList(s map[string]bool) []string {
	list := []string{}
	for item := range s {
		list = append(list, item)
	}
	return list
}

func CompareSets(set1, set2 map[string]bool) (missingInSet1, missingInSet2 []string) {
	// Find items in set2 that are missing in set1
	for item := range set2 {
		if !set1[item] {
			missingInSet1 = append(missingInSet1, item)
		}
	}

	// Find items in set1 that are missing in set2
	for item := range set1 {
		if !set2[item] {
			missingInSet2 = append(missingInSet2, item)
		}
	}

	return missingInSet1, missingInSet2
}

func GetAllKeysFromSyncMap(m *sync.Map) map[string]bool {
	set := make(map[string]bool)

	m.Range(func(key, value interface{}) bool {
		// Assuming keys are of type string
		if strKey, ok := key.(string); ok {
			set[strKey] = true
		}
		return true
	})

	return set
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
