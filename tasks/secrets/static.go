package secrets

import (
	"context"
	"encoding/json"

	runner_tasks "github.com/drone/go-task/task"
	"github.com/sirupsen/logrus"
)

type StaticSecretHandler struct{}

func (h *StaticSecretHandler) Handle(ctx context.Context, req *runner_tasks.Request) runner_tasks.Response {
	var staticSecretSpec StaticSecretSpec
	err := json.Unmarshal(req.Task.Data, &staticSecretSpec)
	logrus.Info("Processing static secrets", *req)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
	}
	// TODO: support batch secrets
	secretResponse := &runner_tasks.Secret{}
	if len(staticSecretSpec.Secrets) > 0 {
		secretResponse.Value = staticSecretSpec.Secrets[0].Value
	}
	return runner_tasks.Respond(secretResponse)
}

type StaticSecret struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

type StaticSecretSpec struct {
	Secrets []*StaticSecret `json:"secrets"`
}
