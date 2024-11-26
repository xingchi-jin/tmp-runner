package vm

import (
	"context"
	"encoding/json"

	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/drone-runners/drone-runner-aws/command/harness"
	"github.com/drone-runners/drone-runner-aws/metric"
	"github.com/drone-runners/drone-runner-aws/store"
	"github.com/drone-runners/drone-runner-aws/types"
	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine/spec"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/tasks/local/utils"
	"github.com/sirupsen/logrus"
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
	metrics         *metric.Metrics
}

func NewSetupHandler(
	taskContext *delegate.TaskContext,
	poolManager drivers.IManager,
	stageOwnerStore store.StageOwnerStore,
	metrics *metric.Metrics,
) *SetupHandler {
	return &SetupHandler{
		taskContext:     taskContext,
		poolManager:     poolManager,
		stageOwnerStore: stageOwnerStore,
		metrics:         metrics,
	}
}

func (h *SetupRequest) Sanitize() {
	h.Network.ID = utils.Sanitize(h.Network.ID)
}

func (h *SetupHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	setupRequest := new(SetupRequest)
	err := json.Unmarshal(req.Task.Data, setupRequest)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
		return task.Error(err)
	}
	setupRequest.Sanitize()
	secrets := []string{}
	for _, v := range req.Secrets {
		secrets = append(secrets, *&v.Value)
	}
	var logConfig api.LogConfig
	var key string
	if req.Task.Logger != nil {
		key = req.Task.Logger.Key
		logConfig = api.LogConfig{
			AccountID: req.Task.Logger.Account,
			URL:       req.Task.Logger.Address,
			Token:     req.Task.Logger.Token,
		}
	}
	setupReq := api.SetupRequest{
		Network:   setupRequest.Network,
		Volumes:   setupRequest.Volumes,
		Envs:      setupRequest.Envs,
		Secrets:   secrets,
		LogConfig: logConfig,
	}

	setupVmRequest := &harness.SetupVMRequest{
		ID:                  setupRequest.Metadata.StageRuntimeID,
		PoolID:              setupRequest.VMConfig.PoolID,
		FallbackPoolIDs:     setupRequest.VMConfig.FallbackPoolIDs,
		ResourceClass:       setupRequest.VMConfig.ResourceClass,
		Tags:                setupRequest.VMConfig.Tags,
		LogKey:              key,
		CorrelationID:       req.Task.ID,
		Context:             convertMetadata(setupRequest.Metadata),
		GitspaceAgentConfig: convertGitspaceConfig(setupRequest.VMConfig.GitspaceAgentConfig),
		StorageConfig:       convertStorageConfig(setupRequest.VMConfig.StorageConfig),
		SetupRequest:        setupReq,
	}

	setupResp, selectedPoolDriver, err := harness.HandleSetup(
		ctx, setupVmRequest, h.stageOwnerStore, []string{}, h.taskContext.PoolMapperByAccount,
		h.taskContext.DelegateName, false, 0, h.poolManager, h.metrics)
	if err != nil {
		return task.Respond(failedResponse(err.Error()))
	}
	var delegateID string
	if h.taskContext.DelegateId != nil {
		delegateID = *h.taskContext.DelegateId
	}
	// Construct final response
	resp := VMTaskExecutionResponse{
		IPAddress:              setupResp.IPAddress,
		CommandExecutionStatus: Success,
		DelegateMetaInfo: DelegateMetaInfo{
			ID: delegateID,
		},
		PoolDriverUsed:        selectedPoolDriver,
		GitspacesPortMappings: setupResp.GitspacesPortMappings,
	}
	return task.Respond(resp)
}

func convertGitspaceConfig(gitspaceAgentConfig GitspaceAgentConfig) types.GitspaceAgentConfig {
	return types.GitspaceAgentConfig{
		Secret:       gitspaceAgentConfig.Secret,
		AccessToken:  gitspaceAgentConfig.AccessToken,
		Ports:        gitspaceAgentConfig.Ports,
		VMInitScript: gitspaceAgentConfig.VMInitScript,
	}
}

func convertStorageConfig(storageConfig StorageConfig) types.StorageConfig {
	return types.StorageConfig{
		CephPoolIdentifier: storageConfig.CephPoolIdentifier,
		Identifier:         storageConfig.Identifier,
		Size:               storageConfig.Size,
	}
}
