package server

import (
	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/harness/runner/delegateshell"
)

// System stores high level System sub-routines.
type System struct {
	delegate    *delegateshell.DelegateShell
	poolManager drivers.IManager
}

func NewSystem(
	delegate *delegateshell.DelegateShell,
	poolManager drivers.IManager,
) *System {
	return &System{
		delegate:    delegate,
		poolManager: poolManager,
	}
}
