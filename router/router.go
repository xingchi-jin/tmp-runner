// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package router

import (
	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/downloader"
	"github.com/drone/go-task/task/drivers/cgi"
	"github.com/harness/runner/delegateshell/daemonset"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/logger/logstream"
	"github.com/harness/runner/tasks/daemontask"
	"github.com/harness/runner/tasks/delegatetask"
	"github.com/harness/runner/tasks/local"
	"github.com/harness/runner/tasks/secrets"
	"github.com/harness/runner/tasks/secrets/vault"
)

func convert(config *delegate.Config) *delegate.TaskContext {
	return &delegate.TaskContext{
		AccountID:              config.Delegate.AccountID,
		Token:                  config.GetToken(),
		DelegateId:             config.Delegate.ID,
		DelegateTaskServiceURL: config.Delegate.TaskServiceURL,
		RunnerType:             config.GetRunnerType(),
		SkipVerify:             config.Server.Insecure,
		ManagerEndpoint:        config.GetHarnessUrl(),
	}
}

func NewRouter(
	taskContext *delegate.TaskContext,
	d downloader.Downloader,
	dsManager *daemonset.DaemonSetManager,
) *task.Router {
	r := task.NewRouter()
	r.Use(logstream.Middleware())

	r.Register("local_init", local.NewSetupHandler(taskContext))
	r.RegisterFunc("local_execute", local.ExecHandler)
	r.RegisterFunc("local_cleanup", local.DestroyHandler)
	r.RegisterFunc("secret/vault/fetch", vault.FetchHandler)
	r.RegisterFunc("secret/vault/edit", vault.Handler)
	r.Register("delegate_task", delegatetask.NewDelegateTaskHandler(taskContext))
	r.Register("secret/static", new(secrets.StaticSecretHandler))

	daemonSetTaskHandler := daemontask.NewDaemonSetTaskHandler(dsManager)
	r.RegisterFunc("daemonset/upsert", daemonSetTaskHandler.HandleUpsert)
	r.RegisterFunc("daemonset/tasks/assign", daemonSetTaskHandler.HandleTaskAssign)

	r.NotFound(cgi.New(d))
	return r
}
