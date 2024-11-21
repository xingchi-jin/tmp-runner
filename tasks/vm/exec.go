package vm

import (
	"context"

	"github.com/drone-runners/drone-runner-aws/app/drivers"
	"github.com/drone-runners/drone-runner-aws/store"
	"github.com/drone/go-task/task"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/tasks/local"
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
}

func NewExecHandler(
	taskContext *delegate.TaskContext,
	poolManager drivers.IManager,
	stageOwnerStore store.StageOwnerStore,
) *ExecHandler {
	return &ExecHandler{
		taskContext:     taskContext,
		poolManager:     poolManager,
		stageOwnerStore: stageOwnerStore,
	}
}

func (h *ExecHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	return nil
}
