// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package logstream

import (
	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/logstream"
	"github.com/harness/lite-engine/pipeline/runtime"
)

// LogWriter creates a log client (`logstream.Writer`) that can be used to write logs.
// NOTE: The caller is responsible for opening (`.Open()`) and
// closing (`.Close()`) the writer after usage is done.
func LogWriter(req *task.Request) logstream.Writer {
	cfg := api.LogConfig{}
	var key string
	if req.Task != nil && req.Task.Logger != nil {
		cfg.AccountID = req.Task.Logger.Account
		cfg.Token = req.Task.Logger.Token
		cfg.URL = req.Task.Logger.Address
		cfg.IndirectUpload = true
		key = req.Task.Logger.Key
	}
	secrets := []string{}
	for _, v := range req.Secrets {
		secrets = append(secrets, v)
	}
	return runtime.GetReplacer(cfg, key, req.Task.ID, secrets)
}
