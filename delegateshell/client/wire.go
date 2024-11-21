package client

import (
	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/delegate"
)

var WireSet = wire.NewSet(
	ProvideManagerClient,
)

func ProvideManagerClient(
	config *delegate.Config,
) Client {
	return NewManagerClient(
		config.GetHarnessUrl(),
		config.Delegate.AccountID,
		config.GetToken(),
		config.Server.Insecure,
		"", // no additional certs directory for now
	)
}
