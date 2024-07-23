package vault

import (
	"context"
	"encoding/json"

	"github.com/drone/go-task/task"
	"github.com/sirupsen/logrus"
)

type DeleteSpec struct {
	Config     *Config `json:"config"`
	EngineName string  `json:"engine_name"`
	Key        string  `json:"key"`
	Path       string  `json:"path"`
}

func DeleteHandler(ctx context.Context, req *task.Request) task.Response {
	in := new(DeleteSpec)

	err := json.Unmarshal(req.Task.Data, in)
	if err != nil {
		return task.Error(err)
	}

	client, err := New(in.Config)
	if err != nil {
		return task.Error(err)
	}

	path := GetFullPathForDelete(in.EngineName, in.Path)
	logrus.Infof("Deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	_, err = client.Logical().Delete(path)
	if err != nil {
		logrus.WithError(err).Errorf("Failed deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
		return task.Error(err)
	}
	logrus.Infof("Done deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	return task.Respond(SecretTaskResponse{})
}
