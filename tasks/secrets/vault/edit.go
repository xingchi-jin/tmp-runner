package vault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type SecretTaskSpec struct {
	Action  string  `json:"action"`
	Config  *Config `json:"config"`
	Key     string  `json:"key"`
	OldPath string  `json:"oldPath"`
	Path    string  `json:"path"`
	Value   string  `json:"value"`
}

type SecretTaskResponse struct {
	CommandExecutionStatus api.CommandExecutionStatus `json:"command_execution_status,omitempty"`
}

func Handler(ctx context.Context, req *task.Request) task.Response {
	in := new(SecretTaskSpec)

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
	case "UPDATE":
		resp, err = HandleUpdate(in, client)
	case "RENAME":
		resp, err = HandleRename(in, client)
	case "DELETE":
		resp, err = HandleDelete(in, client)
	case "VALIDATE":
		resp, err = HandleValidate(in, client)
	default:
		panic(fmt.Sprintf("Unsupported secret task action: %s", action))
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	return task.Respond(respBytes)
}

func HandleCreate(in *SecretTaskSpec, client *vault.Client) (api.VMTaskExecutionResponse, error) {
	return Upsert(*in.Config.SecretEngineName, in.Path, in.Key, in.Value, in.OldPath, false, client)
}

func HandleUpdate(in *SecretTaskSpec, client *vault.Client) (api.VMTaskExecutionResponse, error) {
	return Upsert(*in.Config.SecretEngineName, in.Path, in.Key, in.Value, in.OldPath, false, client)
}

func HandleRename(in *SecretTaskSpec, client *vault.Client) (api.VMTaskExecutionResponse, error) {
	logrus.Infof("Renaming secret value in Vault. Url: [%s]; Path: [%s]; OldPath: [%s]", client.Address(), in.Path, in.OldPath)
	secret, err := Fetch(*in.Config.SecretEngineName, in.OldPath, in.Key, client)
	if err != nil {
		logrus.WithError(err).Errorf("Failed renaming secret value in Vault. Url: [%s]; Path: [%s]", client.Address(), in.OldPath)
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	in.Value = secret
	return Upsert(*in.Config.SecretEngineName, in.Path, in.Key, in.Value, in.OldPath, true, client)
}

func HandleDelete(in *SecretTaskSpec, client *vault.Client) (api.VMTaskExecutionResponse, error) {
	path := GetFullPathForDelete(*in.Config.SecretEngineName, in.Path)
	logrus.Infof("Deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	_, err := client.Logical().Delete(path)
	if err != nil {
		logrus.WithError(err).Errorf("Failed deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	logrus.Infof("Done deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success}, nil
}

func HandleValidate(in *SecretTaskSpec, client *vault.Client) (api.VMTaskExecutionResponse, error) {
	logrus.Infof("Validating secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
	_, err := Fetch(*in.Config.SecretEngineName, in.Path, in.Key, client)
	if err != nil {
		logrus.WithError(err).Errorf("Failed validating secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	logrus.Infof("Done validating secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
	return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success}, nil
}

func Upsert(secretEngineName string, path string, key string, value string, oldPath string, deleteOldPath bool, client *vault.Client) (api.VMTaskExecutionResponse, error) {
	data := map[string]any{
		"data": map[string]string{
			key: value,
		},
	}
	fullPath := GetFullPath(secretEngineName, path)
	logrus.Infof("Writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	_, err := client.Logical().Write(fullPath, data)
	if err != nil {
		logrus.WithError(err).Errorf("Failed writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	logrus.Infof("Done writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	if deleteOldPath {
		fullOldPath := GetFullPathForDelete(secretEngineName, oldPath)
		logrus.Infof("Deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)
		_, err = client.Logical().Delete(fullOldPath)
		if err != nil {
			logrus.WithError(err).Errorf("Failed deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)
		}
		logrus.Infof("Done deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)

	}
	return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success}, nil
}

func Fetch(secretEngineName string, path string, key string, client *vault.Client) (string, error) {
	logrus.Infof("Fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	fullPath := GetFullPath(secretEngineName, path)
	secret, err := client.Logical().Read(fullPath)
	if err != nil {
		logrus.WithError(err).Errorf("Failed fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
		return "", err
	}
	if secret == nil || secret.Data == nil {
		err = errors.New("Could not find secret.")
		logrus.WithError(err).Errorf("Failed fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)

	}

	v := secret.Data["data"]
	if data, ok := v.(map[string]interface{}); ok {
		secret.Data = data
	}

	for k, v := range secret.Data {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if k == key {
			logrus.Infof("Done fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
			return s, nil
		}
	}
	err = errors.New(fmt.Sprintf("Could not find key [%s] in secret data.", key))
	logrus.WithError(err).Errorf("Failed fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	return "", err
}

func GetFullPath(secretEngineName string, path string) string {
	return fmt.Sprintf("%s/data/%s", secretEngineName, path)
}

func GetFullPathForDelete(secretEngineName string, path string) string {
	return fmt.Sprintf("%s/metadata/%s", secretEngineName, path)
}
