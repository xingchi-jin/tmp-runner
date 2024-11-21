package router

import (
	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/downloader"
	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/daemonset"
	"github.com/harness/runner/delegateshell/delegate"
)

var WireSet = wire.NewSet(
	ProvideRouter,
)

func ProvideRouter(
	config *delegate.Config,
	d downloader.Downloader,
	dsManager *daemonset.DaemonSetManager,
) *task.Router {
	return NewRouter(convert(config), d, dsManager)
}
