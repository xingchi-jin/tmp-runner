package router

import (
	"context"

	"github.com/drone/go-task/task"
)

type Router interface {
	Handle(context.Context, *task.Request) task.Response
	WaitForAll()
}
