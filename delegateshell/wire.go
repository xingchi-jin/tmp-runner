package delegateshell

import (
	"path/filepath"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/cloner"
	"github.com/drone/go-task/task/downloader"
	"github.com/drone/go-task/task/packaged"
	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/daemonset"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
)

var WireSet = wire.NewSet(
	ProvideDelegateShell,
	ProvideDownloader,
	ProvidePackageLoader,
)

func ProvideDownloader(config *delegate.Config) (downloader.Downloader, error) {
	return downloader.New(cloner.Default(), filepath.Join(config.CacheLocation, "download")), nil
}

func ProvidePackageLoader(config *delegate.Config) (packaged.PackageLoader, error) {
	return packaged.New(filepath.Join(config.CacheLocation, "default")), nil
}

func ProvideDelegateShell(
	config *delegate.Config,
	managerClient client.Client,
	router *task.Router,
	daemonSetManager *daemonset.DaemonSetManager,
	daemonSetReconciler *daemonset.DaemonSetReconciler,
	downloader downloader.Downloader,
	poller *poller.Poller,
	keepAlive *heartbeat.KeepAlive,
) *DelegateShell {
	return NewDelegateShell(
		config,
		managerClient,
		router,
		daemonSetManager,
		daemonSetReconciler,
		downloader,
		poller,
		keepAlive,
	)
}
