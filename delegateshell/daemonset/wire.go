package daemonset

import (
	"context"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/downloader"
	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/delegate"
)

var WireSet = wire.NewSet(
	ProvideDaemonSetManager,
	ProvideDaemonSetReconciler,
)

func ProvideDaemonSetManager(
	config *delegate.Config,
	downloader downloader.Downloader,
) *DaemonSetManager {
	return NewDaemonSetManager(downloader,
		delegate.IsK8sRunner(config.GetRunnerType()),
		config.Delegate.AccountID,
		config.GetHarnessUrl(),
		config.GetToken(),
		config.EnableRemoteLogging,
		config.Server.Insecure)
}

func ProvideDaemonSetReconciler(
	daemonSetManager *DaemonSetManager,
	router *task.Router,
	managerClient client.Client,
) *DaemonSetReconciler {
	return NewDaemonSetReconciler(
		context.Background(), // TODO: This should probably come from global context, need to verify
		daemonSetManager,
		router,
		managerClient,
	)
}
