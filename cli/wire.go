//go:build wireinject
// +build wireinject

package cli

import (
	"context"

	"github.com/google/wire"
	"github.com/harness/runner/cli/server"
	"github.com/harness/runner/delegateshell"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/daemonset"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
	vmmetrics "github.com/harness/runner/delegateshell/vm/metrics"
	"github.com/harness/runner/delegateshell/vm/pool"
	"github.com/harness/runner/delegateshell/vm/store"
	metricsinjection "github.com/harness/runner/metrics/injection"
	"github.com/harness/runner/router"
)

func initSystem(ctx context.Context, config *delegate.Config) (*server.System, error) {
	wire.Build(
		server.NewSystem,
		daemonset.WireSet,
		router.WireSet,
		delegateshell.WireSet,
		client.WireSet,
		poller.WireSet,
		heartbeat.WireSet,
		metricsinjection.WireSet,

		// Dependencies required for managing VMs.
		pool.WireSet,
		store.WireSet,
		vmmetrics.WireSet,
	)
	return &server.System{}, nil
}
