package vault

import (
	"context"
	"encoding/json"

	"github.com/drone/go-task/task"
)

type UpdateSpec struct {
	Config     *Config `json:"config"`
	EngineName string  `json:"engine_name"`
	Key        string  `json:"key"`
	Path       string  `json:"path"`
	Value      string  `json:"value"`
}

func UpdateHandler(ctx context.Context, req *task.Request) task.Response {
	in := new(UpdateSpec)

	err := json.Unmarshal(req.Task.Data, in)
	if err != nil {
		return task.Error(err)
	}

	client, err := New(in.Config)
	if err != nil {
		return task.Error(err)
	}

	err = Upsert(in.EngineName, in.Path, in.Key, in.Value, "", false, client)
	if err != nil {
		return task.Error(err)
	}

	return task.Respond(SecretTaskResponse{})
}
