package router

import (
	"context"

	runner_tasks "github.com/drone/go-task/task"
)

type Router interface {
	Handle(context.Context, *runner_tasks.Request) runner_tasks.Response
	WaitForAll()
}
