package vault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	vault "github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

type EditVaultSecretSpec struct {
	Config *Config `json:"config"`
	Path   string  `json:"path"`
	Key    string  `json:"key"`
	Value  string  `json:"value"`
	Action string  `json:"action"`
}

type SecretTaskResponse struct {
	CommandExecutionStatus api.CommandExecutionStatus `json:"command_execution_status,omitempty"`
}

func Handler(ctx context.Context, req *task.Request) task.Response {
	in := new(EditVaultSecretSpec)

	// decode the task input.
	err := json.Unmarshal(req.Task.Data, in)
	if err != nil {
		return task.Error(err)
	}

	client, err := New(in.Config)
	if err != nil {
		return task.Error(err)
	}

	var resp api.VMTaskExecutionResponse
	switch action := in.Action; action {
	case "CREATE":
		resp, err = HandleCreate(in, client)
	default:
		panic(fmt.Sprintf("Unsupported secret action: %s", action))

	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	return task.Respond(respBytes)
}

func HandleCreate(in *EditVaultSecretSpec, client *vault.Client) (api.VMTaskExecutionResponse, error) {
	data := map[string]any{
		"data": map[string]string{
			in.Key: in.Value,
		},
	}
	logrus.Infof("Writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
	_, err := client.Logical().Write(in.Path, data)
	if err != nil {
		logrus.WithError(err).Errorf("Failed writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	logrus.Infof("Done writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
	return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success}, nil
}
