package logstream

import (
	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/logstream"
	"github.com/harness/lite-engine/pipeline/runtime"
)

// Returns a `logstream.Writer`, which can either be a custom logger
// (`runtime.GetReplacer`) or a `stdoutWriter`.
// NOTE: The caller is responsible for closing the writer (`.Close()`)
// after usage is done.
func GetLogstreamWriter(req *task.Request) logstream.Writer {
	// if a logger has been provided in the task which points to a custom endpoint,
	// we create a custom writer
	// NOTE: the caller is responsible for closing the writer after usage is done.
	if req.Task != nil && req.Task.Logger != nil && req.Task.Logger.Address != "" {
		writer := logWriter(req)
		writer.Open()
		return writer
	} else {
		// write logs to stdout if custom logger is not provided.
		return newStdoutWriter()
	}
}

// logWriter creates a log client (`logstream.Writer`) that can be used to write logs.
func logWriter(req *task.Request) logstream.Writer {
	cfg := api.LogConfig{}
	cfg.AccountID = req.Task.Logger.Account
	cfg.Token = req.Task.Logger.Token
	cfg.URL = req.Task.Logger.Address
	cfg.IndirectUpload = true
	key := req.Task.Logger.Key
	secrets := []string{}
	for _, v := range req.Secrets {
		secrets = append(secrets, v)
	}
	return runtime.GetReplacer(cfg, key, req.Task.ID, secrets)
}
