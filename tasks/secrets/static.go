package secrets

import (
	"context"
	"encoding/json"

	"github.com/drone/go-task/task"
	"github.com/sirupsen/logrus"
)

type StaticSecretHandler struct{}

func (h *StaticSecretHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	var staticSecretSpec StaticSecretSpec
	err := json.Unmarshal(req.Task.Data, &staticSecretSpec)
	logrus.Info("Processing static secrets", *req)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
	}
	// TODO: support batch secrets
	secretResponse := &task.Secret{}
	if len(staticSecretSpec.Secrets) > 0 {
		secretResponse.Value = staticSecretSpec.Secrets[0].Value
	}
	return task.Respond(secretResponse)
}

type StaticSecret struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

type StaticSecretSpec struct {
	Secrets []*StaticSecret `json:"secrets"`
}
