package vm

import (
	"context"

	"github.com/drone/go-task/task"
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
}

func NewCleanupHandler() *CleanupHandler {
	return &CleanupHandler{}
}

func (h *CleanupHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	return nil
}
