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
	)
	return &server.System{}, nil
}
