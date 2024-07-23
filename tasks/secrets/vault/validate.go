package vault

import (
	"context"
	"encoding/json"

	"github.com/drone/go-task/task"
	"github.com/sirupsen/logrus"
)

type ValidateSpec struct {
	Config     *Config `json:"config"`
	EngineName string  `json:"engine_name"`
	Key        string  `json:"key"`
	Path       string  `json:"path"`
}

func ValidateHandler(ctx context.Context, req *task.Request) task.Response {
	in := new(CreateSpec)

	err := json.Unmarshal(req.Task.Data, in)
	if err != nil {
		return task.Error(err)
	}

	client, err := New(in.Config)
	if err != nil {
		return task.Error(err)
	}

	logrus.Infof("Validating secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
	_, err = Fetch(in.EngineName, in.Path, in.Key, client)
	if err != nil {
		logrus.WithError(err).Errorf("Failed validating secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return task.Error(err)
	}
	return task.Respond(SecretTaskResponse{})
}
