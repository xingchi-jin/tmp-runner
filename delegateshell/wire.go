package delegateshell

import (
	"os"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/cloner"
	"github.com/drone/go-task/task/downloader"
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
)

func ProvideDownloader() (downloader.Downloader, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return downloader.Downloader{}, err
	}
	return downloader.New(cloner.Default(), cache), nil
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
