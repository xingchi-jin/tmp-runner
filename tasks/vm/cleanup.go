package vm

import (
	"context"
	"encoding/json"

	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/drone-runners/drone-runner-aws/command/harness"
	"github.com/drone-runners/drone-runner-aws/command/harness/storage"
	"github.com/drone-runners/drone-runner-aws/metric"
	"github.com/drone-runners/drone-runner-aws/store"
	"github.com/drone/go-task/task"
	"github.com/sirupsen/logrus"
)

type CleanupType string

const (
	Detach CleanupType = "DETACH"
	Delete CleanupType = "DELETE"
)

type CleanupRequest struct {
	Metadata Metadata  `json:"metadata"`
	Instance *Instance `json:"instance"`
}

type Instance struct {
	ID                 string      `json:"id"`
	StorageCleanupType CleanupType `json:"storage_cleanup_type"`
}

type CleanupHandler struct {
	poolManager     drivers.IManager
	stageOwnerStore store.StageOwnerStore
	metrics         *metric.Metrics
}

func NewCleanupHandler(
	poolManager drivers.IManager,
	stageOwnerStore store.StageOwnerStore,
	metrics *metric.Metrics,
) *CleanupHandler {
	return &CleanupHandler{
		poolManager:     poolManager,
		stageOwnerStore: stageOwnerStore,
		metrics:         metrics,
	}
}

func (h *CleanupHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	cleanupRequest := new(CleanupRequest)
	err := json.Unmarshal(req.Task.Data, cleanupRequest)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
		return task.Error(err)
	}
	var cleanupType storage.CleanupType
	if cleanupRequest.Instance != nil {
		cleanupType = convertCleanupType(cleanupRequest.Instance.StorageCleanupType)
	}
	var logKey string
	if req.Task.Logger != nil {
		logKey = req.Task.Logger.Key
	}
	cleanupReq := &harness.VMCleanupRequest{
		Context:            convertMetadata(cleanupRequest.Metadata),
		StageRuntimeID:     cleanupRequest.Metadata.StageRuntimeID,
		LogKey:             logKey,
		Distributed:        true,
		StorageCleanupType: cleanupType,
	}

	err = harness.HandleDestroy(ctx, cleanupReq, h.stageOwnerStore, false, 0, h.poolManager, h.metrics)
	if err != nil {
		logrus.Error("could not handle cleanup request: %w", err)
		return task.Error(err)
	}
	return task.Respond(nil)
}

func convertCleanupType(t CleanupType) storage.CleanupType {
	switch t {
	case Detach:
		return storage.Detach
	case Delete:
		return storage.Delete
	default:
		return storage.Delete
	}
}
