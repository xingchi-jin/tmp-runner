// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package logstream

import (
	"testing"

	"github.com/drone/go-task/task"
)

func TestGetLogstreamWriter(t *testing.T) {
	reqWithoutLogger := &task.Request{Task: &task.Task{}}
	logger := GetLogstreamWriter(reqWithoutLogger)
	logger.Close()
	_, ok := logger.(*stdoutWriter)
	if !ok {
		t.Error("Want logger type stdoutWriter, not replacer")
	}
}
