package secrets

import (
	"context"
	"encoding/base64"
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
		secret := staticSecretSpec.Secrets[0]
		secretValue := secret.Value
		if secret.Base64 == true {
			decodedValue, err := base64.StdEncoding.DecodeString(secretValue)
			if err != nil {
				logrus.Error("Error occurred when decoding base64 secret. %w", err)
				return task.Respond(secretResponse)
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
