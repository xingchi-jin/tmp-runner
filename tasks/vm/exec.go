package vm

import (
	"context"
	"encoding/json"
	"time"

	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/drone-runners/drone-runner-aws/command/harness"
	"github.com/drone-runners/drone-runner-aws/metric"
	"github.com/drone-runners/drone-runner-aws/store"
	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/tasks/local"
	"github.com/harness/runner/tasks/local/utils"
	"github.com/sirupsen/logrus"
)

var (
	audience          = "audience"
	issuer            = "issuer"
	tokenExpiryOffset = 30 * time.Minute
	// in hosted, the VMs directly send step responses to the manager.
	// to enable easy local testing, async can be set to false to make the response
	// sync instead of having to start up the manager to get the response back.
	async = true
)

type ExecRequest struct {
	Request  local.ExecRequest `json:"request"` // reuse local exec request
	Forward  *Forward          `json:"forward"`
	Metadata Metadata          `json:"metadata"`
}

// Forward represents the information about the VM where the request needs
// to be forwarded.
type Forward struct {
	ID        string `json:"id"`
	IPAddress string `json:"ip_address"`
	Certs     *Certs `json:"certs"`
}

type Certs struct {
	Public  []byte `json:"public"`
	Private []byte `json:"private"`
	CA      []byte `json:"ca"`
}

type ExecHandler struct {
	taskContext     *delegate.TaskContext
	stageOwnerStore store.StageOwnerStore
	poolManager     drivers.IManager
	metrics         *metric.Metrics
}

func NewExecHandler(
	taskContext *delegate.TaskContext,
	poolManager drivers.IManager,
	stageOwnerStore store.StageOwnerStore,
	metrics *metric.Metrics,
) *ExecHandler {
	return &ExecHandler{
		taskContext:     taskContext,
		poolManager:     poolManager,
		stageOwnerStore: stageOwnerStore,
		metrics:         metrics,
	}
}

func (e *ExecRequest) Sanitize() {
	e.Request.ID = utils.Sanitize(e.Request.ID)
	e.Request.Network = utils.Sanitize(e.Request.Network)
}

func (h *ExecHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	execRequest := new(ExecRequest)
	err := json.Unmarshal(req.Task.Data, execRequest)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
		return task.Error(err)
	}
	execRequest.Sanitize()
	if req.Task.Logger != nil {
		execRequest.Request.LogKey = req.Task.Logger.Key
	}
	secrets := []string{}
	for _, v := range req.Secrets {
		secrets = append(secrets, *&v.Value)
	}
	execRequest.Request.Secrets = secrets
	// Generate a token so that the task can send back the response back to the manager directly
	token, err := delegate.Token(audience, issuer, h.taskContext.AccountID, h.taskContext.Token, 10*time.Hour+tokenExpiryOffset)
	if err != nil {
		return task.Respond(failedResponse(err.Error()))
	}

	var delegateID string
	if h.taskContext.DelegateId != nil {
		delegateID = *h.taskContext.DelegateId
	}
	execRequest.Request.StepStatus = api.StepStatusConfig{
		Endpoint:   h.taskContext.ManagerEndpoint,
		AccountID:  h.taskContext.AccountID,
		TaskID:     req.Task.ID,
		DelegateID: delegateID,
		Token:      token,
		// RunnerResponse: execRequest.Request.StepStatus.RunnerResponse, ---- what is this needed for?
	}

	execVMRequest := &harness.ExecuteVMRequest{
		StageRuntimeID:   execRequest.Metadata.StageRuntimeID,
		InstanceID:       execRequest.Forward.ID,
		IPAddress:        execRequest.Forward.IPAddress,
		Distributed:      true,
		CorrelationID:    req.Task.ID,
		StartStepRequest: execRequest.Request.StartStepRequest,
	}

	pollStepResp, err := harness.HandleStep(ctx, execVMRequest, h.stageOwnerStore, []string{}, false, 0, h.poolManager, h.metrics, async)
	if err != nil {
		return task.Respond(failedResponse(err.Error()))
	}
	var resp VMTaskExecutionResponse
	if async {
		if pollStepResp.Error == "" {
			// Do not send the response for a successful submit of the vm_execute task. The response will be sent by lite engine.
			return nil
		} else {
			resp = VMTaskExecutionResponse{CommandExecutionStatus: Failure, ErrorMessage: pollStepResp.Error}
		}
		return task.Respond(resp)
	}

	// Construct final response
	resp = convert(pollStepResp)
	resp.DelegateMetaInfo.ID = delegateID
	// TODO: Set delegate meta info host name as well
	return task.Respond(resp)
}

// convert poll response to a Vm task execution response
func convert(r *api.PollStepResponse) VMTaskExecutionResponse {
	if r.Error == "" {
		return VMTaskExecutionResponse{CommandExecutionStatus: Success, OutputVars: r.Outputs, Artifact: r.Artifact, Outputs: convertOutputs(r.OutputV2), OptimizationState: r.OptimizationState}
	}
	return VMTaskExecutionResponse{CommandExecutionStatus: Failure, ErrorMessage: r.Error, OptimizationState: r.OptimizationState}
}

func convertOutputs(outputs []*api.OutputV2) []*OutputV2 {
	var convertedOutputs []*OutputV2
	for _, output := range outputs {
		convertedOutputs = append(convertedOutputs, &OutputV2{
			Key:   output.Key,
			Value: output.Value,
			Type:  OutputType(output.Type),
		})
	}
	return convertedOutputs
}
