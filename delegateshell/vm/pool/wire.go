package pool

import (
	"context"

	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/drone-runners/drone-runner-aws/store"
	"github.com/drone-runners/drone-runner-aws/types"
	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/delegate"
)

var WireSet = wire.NewSet(
	ProvideManager,
)

// ProvideManager is a Wire provider function that returns a new pool manager.
func ProvideManager(
	ctx context.Context,
	instanceStore store.InstanceStore,
	stageOwnerStore store.StageOwnerStore,
	config *delegate.Config,
) drivers.IManager {
	if config.VM.Pool.File == "" {
		return nil
	}
	manager := drivers.NewManager(
		ctx,
		instanceStore,
		stageOwnerStore,
		types.Tmate{},
		config.Delegate.Name,
		config.VM.BinaryURI.LiteEngine,
		config.VM.BinaryURI.SplitTests,
		config.VM.BinaryURI.Plugin,
		config.VM.BinaryURI.AutoInjection,
	)
	return drivers.NewDistributedManager(manager)
}
