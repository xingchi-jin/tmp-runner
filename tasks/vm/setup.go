package vm

import (
	"context"

	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/drone-runners/drone-runner-aws/store"
	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/engine/spec"
	"github.com/harness/runner/delegateshell/delegate"
)

type SetupRequest struct {
	Network  spec.Network      `json:"network"`
	Volumes  []*spec.Volume    `json:"volumes"`
	Envs     map[string]string `json:"envs"`
	VMConfig VMConfig          `json:"vm_config"`
	Metadata Metadata          `json:"metadata"`
}

type GitspaceAgentConfig struct {
	Secret       string `json:"secret"`       // Deprecated: VMInitScript should be used to send the whole script
	AccessToken  string `json:"access_token"` // Deprecated: VMInitScript should be used to send the whole script
	Ports        []int  `json:"ports"`
	VMInitScript string `json:"vm_init_script"`
}

type StorageConfig struct {
	CephPoolIdentifier string `json:"ceph_pool_identifier"`
	Identifier         string `json:"identifier"`
	Size               string `json:"size"`
}

type VMConfig struct {
	PoolID              string              `json:"pool_id"`
	Tags                map[string]string   `json:"tags"`
	FallbackPoolIDs     []string            `json:"fallback_pool_ids"`
	GitspaceAgentConfig GitspaceAgentConfig `json:"gitspace_agent_config"`
	StorageConfig       StorageConfig       `json:"storage_config"`
	ResourceClass       string              `json:"resource_class"`
}

type SetupHandler struct {
	taskContext     *delegate.TaskContext
	poolManager     drivers.IManager
	stageOwnerStore store.StageOwnerStore
}

func NewSetupHandler(
	taskContext *delegate.TaskContext,
	poolManager drivers.IManager,
	stageOwnerStore store.StageOwnerStore,
) *SetupHandler {
	return &SetupHandler{
		taskContext:     taskContext,
		poolManager:     poolManager,
		stageOwnerStore: stageOwnerStore,
	}
}

func (h *SetupHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	return nil
}
