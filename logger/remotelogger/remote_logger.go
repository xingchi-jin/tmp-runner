package remotelogger

import (
	"context"

	"github.com/harness/runner/logger/remotelogger/gcplogger"

	"github.com/harness/runner/logger"

	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/version"
)

func Start(ctx context.Context, accountId, managerEndpoint, runnerToken, serviceName, entityName string, remoteLoggingEnabled, insecure bool) {
	if !remoteLoggingEnabled {
		logger.Info(ctx, "Not pushing logs to remote. To enable remote logging, set environment variable ENABLE_REMOTE_LOGGING=true")
		return
	}
	managerClient := client.NewManagerClient(managerEndpoint, accountId, runnerToken, insecure, "")

	err := gcplogger.Initialize(ctx, managerClient)
	if err != nil {
		logger.WithError(ctx, err).Error("failed to start gcp logger. Disabling remote logging")
		return
	}

	// applied only to remote logging to avoid cluttering in other log output
	logger.UpdateContextInHooks(map[string]string{
		"accountId":  accountId,
		"managerUrl": managerEndpoint,
		"service":    serviceName,
		"version":    version.Version,
		"name":       entityName})
}
