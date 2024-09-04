// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package router

import (
	"log"
	"os"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/cloner"
	"github.com/drone/go-task/task/download"
	"github.com/drone/go-task/task/drivers/cgi"
	"github.com/harness/runner/daemonset/manager"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/logger/logstream"
	"github.com/harness/runner/tasks/delegatetask"
	"github.com/harness/runner/tasks/local"
	"github.com/harness/runner/tasks/secrets"
	"github.com/harness/runner/tasks/secrets/vault"
)

func NewRouter(taskContext *delegate.TaskContext) *task.Router {
	r := task.NewRouter()
	r.Use(logstream.Middleware())

	r.Register("local_init", local.NewSetupHandler(taskContext))
	r.RegisterFunc("local_execute", local.ExecHandler)
	r.RegisterFunc("local_cleanup", local.DestroyHandler)
	r.RegisterFunc("secret/vault/fetch", vault.FetchHandler)
	r.RegisterFunc("secret/vault/edit", vault.Handler)
	r.Register("delegate_task", delegatetask.NewDelegateTaskHandler(taskContext))
	r.Register("secret/static", new(secrets.StaticSecretHandler))

	cache, err := os.UserCacheDir()
	if err != nil {
		log.Fatalln(err)
	}
	downloader := download.New(cloner.Default(), cache)
	daemonSetManager := manager.New(downloader, delegate.IsK8sRunner(taskContext.RunnerType))
	r.RegisterFunc("daemonset/upsert", daemonSetManager.HandleUpsert)
	r.RegisterFunc("daemonset/tasks/assign", daemonSetManager.HandleTaskAssign)
	r.NotFound(cgi.New(downloader))
	return r
}
