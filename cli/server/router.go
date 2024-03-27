// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package server

import (
	"context"
	"sync"

	runner_tasks "github.com/drone/go-task/task"
	"github.com/harness/runner/tasks"
)

type Router struct {
	router *runner_tasks.Router
	wg     sync.WaitGroup
}

func NewRouter() *Router {
	r := runner_tasks.NewRouter()

	// TODO: handlers should be registered in the go-task lib
	r.Register("local_init", new(tasks.SetupHandler))
	r.Register("local_execute", new(tasks.ExecHandler))
	r.Register("local_cleanup", new(tasks.DestroyHandler))
	return &Router{
		router: r,
	}
}

func (r *Router) WaitForAll() {
	r.wg.Wait()
}

func (r *Router) Handle(ctx context.Context, req *runner_tasks.Request) runner_tasks.Response {
	r.wg.Add(1)
	defer func() {
		r.wg.Done()
	}()
	resp := r.router.Handle(ctx, req)
	return resp
}
