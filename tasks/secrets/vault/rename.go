package vault

import (
	"context"
	"encoding/json"

	"github.com/drone/go-task/task"
	"github.com/sirupsen/logrus"
)

type RenameSpec struct {
	Config     *Config `json:"config"`
	EngineName string  `json:"engine_name"`
	Key        string  `json:"key"`
	OldPath    string  `json:"old_path"`
	Path       string  `json:"path"`
}

func RenameHandler(ctx context.Context, req *task.Request) task.Response {
	in := new(RenameSpec)

	err := json.Unmarshal(req.Task.Data, in)
	if err != nil {
		return task.Error(err)
	}

	client, err := New(in.Config)
	if err != nil {
		return task.Error(err)
	}

	logrus.Infof("Renaming secret value in Vault. Url: [%s]; Path: [%s]; OldPath: [%s]", client.Address(), in.Path, in.OldPath)
	secret, err := Fetch(in.EngineName, in.OldPath, in.Key, client)
	if err != nil {
		logrus.WithError(err).Errorf("Failed renaming secret value in Vault. Url: [%s]; Path: [%s]", client.Address(), in.OldPath)
		return task.Error(err)
	}
	err = Upsert(in.EngineName, in.Path, in.Key, secret, in.OldPath, true, client)
	if err != nil {
		return task.Error(err)
	}

	return task.Respond(SecretTaskResponse{})
}
