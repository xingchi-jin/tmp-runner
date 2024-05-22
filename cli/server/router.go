// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package server

import (
	"context"
	"sync"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/cloner"
	"github.com/drone/go-task/task/download"
	"github.com/drone/go-task/task/drivers/cgi"
	"github.com/harness/runner/tasks"
	"github.com/harness/runner/tasks/secrets"
	"github.com/harness/runner/tasks/secrets/vault"
)

type Router struct {
	router *task.Router
	wg     sync.WaitGroup
}

func NewRouter() *Router {
	r := task.NewRouter()

	// TODO: handlers should be registered in the go-task lib
	r.RegisterFunc("local_init", tasks.SetupHandler)
	r.RegisterFunc("local_execute", tasks.ExecHandler)
	r.RegisterFunc("local_cleanup", tasks.DestroyHandler)
	r.RegisterFunc("secret/vault/fetch", vault.FetchHandler)
	r.Register("secret/static", new(secrets.StaticSecretHandler))
	downloader := download.New(cloner.Default())
	r.NotFound(cgi.New(downloader))
	return &Router{
		router: r,
	}
}

func (r *Router) WaitForAll() {
	r.wg.Wait()
}

func (r *Router) Handle(ctx context.Context, req *task.Request) task.Response {
	r.wg.Add(1)
	defer func() {
		r.wg.Done()
	}()
	resp := r.router.Handle(ctx, req)
	return resp
}
