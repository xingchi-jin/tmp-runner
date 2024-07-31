package vault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/drone/go-task/task"
	vault "github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

type VaultSecretTaskRequest struct {
	Action     string  `json:"action"`
	Config     *Config `json:"config"`
	EngineName string  `json:"engine_name"`
	Key        string  `json:"key"`
	OldPath    string  `json:"old_path"`
	Path       string  `json:"path"`
	Value      string  `json:"value"`
}

type VaultSecretTaskResponse struct{}

func Handler(ctx context.Context, req *task.Request) task.Response {
	in := new(VaultSecretTaskRequest)

	// decode the task input.
	err := json.Unmarshal(req.Task.Data, in)
	if err != nil {
		return task.Error(err)
	}

	client, err := New(in.Config)
	if err != nil {
		return task.Error(err)
	}

	switch action := in.Action; action {
	case "UPSERT":
		return handleUpsert(in, client)
	case "RENAME":
		return handleRename(in, client)
	case "DELETE":
		return handleDelete(in, client)
	case "VALIDATE":
		return handleValidate(in, client)
	default:
		err = fmt.Errorf("unsupported secret task action: %s", action)
		logrus.Error(err)
		return task.Error(err)
	}
}

func handleUpsert(in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	err := upsert(in.EngineName, in.Path, in.Key, in.Value, "", false, client)
	if err != nil {
		logrus.WithError(err).Errorf("failed upserting secret value in Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return task.Error(err)
	}
	return task.Respond(VaultSecretTaskResponse{})
}

func handleRename(in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	logrus.Infof("renaming secret value in Vault. Url: [%s]; Path: [%s]; OldPath: [%s]", client.Address(), in.Path, in.OldPath)
	secret, err := fetch(in.EngineName, in.OldPath, in.Key, client)
	if err != nil {
		logrus.WithError(err).Errorf("failed renaming secret value in Vault. Url: [%s]; Path: [%s]", client.Address(), in.OldPath)
		return task.Error(err)
	}
	err = upsert(in.EngineName, in.Path, in.Key, secret, in.OldPath, true, client)
	if err != nil {
		logrus.WithError(err).Errorf("failed renaming secret value in Vault. Url: [%s]; Path: [%s]", client.Address(), in.OldPath)
		return task.Error(err)
	}

	return task.Respond(VaultSecretTaskResponse{})
}

func handleDelete(in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	path := getFullPathForDelete(in.EngineName, in.Path)
	logrus.Infof("deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	_, err := client.Logical().Delete(path)
	if err != nil {
		logrus.WithError(err).Errorf("failed deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
		return task.Error(err)
	}
	logrus.Infof("done deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	return task.Respond(VaultSecretTaskResponse{})
}

func handleValidate(in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	logrus.Infof("validating secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
	_, err := fetch(in.EngineName, in.Path, in.Key, client)
	if err != nil {
		logrus.WithError(err).Errorf("failed validating secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return task.Error(err)
	}
	return task.Respond(VaultSecretTaskResponse{})
}

func upsert(engineName string, path string, key string, value string, oldPath string, deleteOldPath bool, client *vault.Client) error {
	data := map[string]any{
		"data": map[string]string{
			key: value,
		},
	}
	fullPath := getFullPath(engineName, path)
	logrus.Infof("writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	_, err := client.Logical().Write(fullPath, data)
	if err != nil {
		return err
	}
	logrus.Infof("done writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	if deleteOldPath {
		fullOldPath := getFullPathForDelete(engineName, oldPath)
		logrus.Infof("deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)
		_, err = client.Logical().Delete(fullOldPath)
		if err != nil {
			logrus.WithError(err).Errorf("failed deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)
		}
		logrus.Infof("done deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)

	}
	return nil
}

func fetch(engineName string, path string, key string, client *vault.Client) (string, error) {
	logrus.Infof("fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	fullPath := getFullPath(engineName, path)
	secret, err := client.Logical().Read(fullPath)
	if err != nil {
		return "", err
	}
	if secret == nil || secret.Data == nil {
		err = fmt.Errorf("could not find secret. Url: [%s]; Path: [%s]", client.Address(), fullPath)
		return "", err
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
			logrus.Infof("done fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
			return s, nil
		}
	}
	err = fmt.Errorf("could not find key [%s] in secret data.", key)
	return "", err
}

func getFullPath(engineName string, path string) string {
	return fmt.Sprintf("%s/data/%s", engineName, path)
}

func getFullPathForDelete(secretEngineName string, path string) string {
	return fmt.Sprintf("%s/metadata/%s", secretEngineName, path)
}
