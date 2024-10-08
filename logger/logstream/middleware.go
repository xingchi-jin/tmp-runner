package logstream

import (
	"context"
	"os"

	"github.com/drone/go-task/task"
)

// Middleware sets the request logger to a custom implementation
// that points to the logging service.
func Middleware() func(next task.Handler) task.Handler {
	return func(next task.Handler) task.Handler {
		fn := func(ctx context.Context, req *task.Request) task.Response {
			// If a logger has been provided in the task which points to a custom endpoint,
			// we create a custom writer and feed it into the logger.
			if req.Task != nil && req.Task.Logger != nil && req.Task.Logger.Address != "" {
				writer := LogWriter(req)
				req.Logger = writer
			} else {
				req.Logger = os.Stdout // write logs to stdout if custom logger is not provided.
			}
			return next.Handle(ctx, req)
		}
		return task.HandlerFunc(fn)
	}
}
