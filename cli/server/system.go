package server

import (
	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/harness/runner/delegateshell"
	metricshandler "github.com/harness/runner/metrics/handler"
)

// System stores high level System sub-routines.
type System struct {
	delegate       *delegateshell.DelegateShell
	poolManager    drivers.IManager
	metricsHandler *metricshandler.MetricsHandler
}

func NewSystem(
	delegate *delegateshell.DelegateShell,
	poolManager drivers.IManager,
	metricsHandler *metricshandler.MetricsHandler,
) *System {
	return &System{
		delegate:       delegate,
		poolManager:    poolManager,
		metricsHandler: metricsHandler,
	}
}
