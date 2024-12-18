package poller

import (
	"github.com/drone/go-task/task"
	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/metrics"
)

// WireSet is a Wire provider set that provides a Poller.
var WireSet = wire.NewSet(
	ProvidePoller,
)

// ProvidePoller is a Wire provider function that creates a Poller.
func ProvidePoller(
	client client.Client,
	router *task.Router,
	config *delegate.Config,
	metrics metrics.Metrics,
) *Poller {
	return New(client, router, metrics, config.EnableRemoteLogging)
}
