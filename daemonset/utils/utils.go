// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package utils

import (
	"github.com/harness/runner/daemonset"
	"github.com/sirupsen/logrus"
)

// returns a logrus *Entry with daemon set's data as fields
func DsLogger(ds *daemonset.DaemonSet) *logrus.Entry {
	return logrus.WithField("id", ds.DaemonSetId).
		WithField("type", ds.Type).
		WithField("port", ds.HttpSever.Port).
		WithField("pid", ds.HttpSever.Execution.Process.Pid).
		WithField("binpath", ds.HttpSever.Execution.Path)
}
