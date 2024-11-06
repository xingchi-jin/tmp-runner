package gcplogger

import (
	"context"
	"fmt"

	"github.com/harness/runner/delegateshell/client"

	"github.com/harness/runner/logger"
)

// Initialize GCPLogger https://cloud.google.com/go/docs/reference/cloud.google.com/go/logging/latest
func Initialize(ctx context.Context, managerClient *client.ManagerClient) error {
	tokenManager, err := NewTokenManager(ctx, managerClient)
	if err != nil {
		return fmt.Errorf("failed to initialize token provider: %w", err)
	}

	hook, err := newGcpLoggingHook(ctx, logger.LogFileName, tokenManager.projectID, tokenManager)
	if err != nil {
		return fmt.Errorf("failed to create stack driver hook: %w", err)
	}

	logger.AddHook(hook)
	logger.Infoln("GCP logging enabled")
	return nil
}
