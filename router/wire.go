package router

import (
	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/drone-runners/drone-runner-aws/metric"
	"github.com/drone-runners/drone-runner-aws/store"
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
	poolManager drivers.IManager,
	stageOwnerStore store.StageOwnerStore,
	vmmetrics *metric.Metrics,
) *task.Router {
	return NewRouter(convert(config), d, dsManager, poolManager, stageOwnerStore, vmmetrics)
}
