package secrets

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/harness/runner/logger"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/common"
)

type StaticSecretHandler struct{}

func (h *StaticSecretHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	var staticSecretSpec StaticSecretSpec
	err := json.Unmarshal(req.Task.Data, &staticSecretSpec)
	logger.Info("Processing static secrets", *req)
	if err != nil {
		logger.Error("Error occurred during unmarshalling. %w", err)
	}
	// TODO: support batch secrets
	secretResponse := &common.Secret{}
	if len(staticSecretSpec.Secrets) > 0 {
		secret := staticSecretSpec.Secrets[0]
		secretValue := secret.Value
		if secret.Base64 {
			decodedValue, err := base64.StdEncoding.DecodeString(secretValue)
			if err != nil {
				return task.Error(fmt.Errorf("Error occurred when decoding base64 secret. %w", err))
			}
			secretValue = string(decodedValue)
		}
		secretResponse.Value = secretValue
	}
	return task.Respond(secretResponse)
}

type StaticSecret struct {
	Id     string `json:"id"`
	Value  string `json:"value"`
	Base64 bool   `json:"base64"`
}

type StaticSecretSpec struct {
	Secrets []*StaticSecret `json:"secrets"`
}
