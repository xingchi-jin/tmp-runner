package heartbeat

import (
	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/metrics"
)

// WireSet is a Wire provider set that provides a KeepAlive.
var WireSet = wire.NewSet(
	ProvideKeepAlive,
)

func ProvideKeepAlive(
	config *delegate.Config,
	managerClient client.Client,
	metrics metrics.Metrics,
) *KeepAlive {
	return New(
		config.Delegate.AccountID,
		config.GetName(),
		config.GetTags(),
		config.GetCapacityConfig(),
		managerClient,
		metrics,
	)
}
