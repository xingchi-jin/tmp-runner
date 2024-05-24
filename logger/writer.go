package logger

import (
	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/logstream"
	"github.com/harness/lite-engine/pipeline/runtime"
)

// LogWriter creates a log client that can be used to write logs.
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
