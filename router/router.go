// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package router

import (
	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/cloner"
	"github.com/drone/go-task/task/download"
	"github.com/drone/go-task/task/drivers/cgi"
	"github.com/harness/runner/logger"
	"github.com/harness/runner/tasks/delegatetask"
	"github.com/harness/runner/tasks/local"
	"github.com/harness/runner/tasks/secrets"
	"github.com/harness/runner/tasks/secrets/vault"
)

func NewRouter() *task.Router {
	r := task.NewRouter()
	r.Use(logger.Middleware())

	r.RegisterFunc("local_init", local.SetupHandler)
	r.RegisterFunc("local_execute", local.ExecHandler)
	r.RegisterFunc("local_cleanup", local.DestroyHandler)
	r.RegisterFunc("secret/vault/fetch", vault.FetchHandler)
	r.RegisterFunc("delegate_task", delegatetask.DelegateTaskHandler)
	r.Register("secret/static", new(secrets.StaticSecretHandler))
	downloader := download.New(cloner.Default())
	r.NotFound(cgi.New(downloader))

	return r
}
