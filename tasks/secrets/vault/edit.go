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
	Action        string  `json:"action"`
	Config        *Config `json:"config"`
	EngineName    string  `json:"engine_name"`
	EngineVersion uint8   `json:"engine_version"`
	Key           string  `json:"key"`
	Path          string  `json:"path"`
	Value         string  `json:"value"`
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
	case "DELETE":
		return handleDelete(in, client)
	default:
		err = fmt.Errorf("unsupported secret task action: %s", action)
		logrus.Error(err)
		return task.Error(err)
	}
}

func handleUpsert(in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	err := upsert(in.EngineVersion, in.EngineName, in.Path, in.Key, in.Value, client)
	if err != nil {
		logrus.WithError(err).Errorf("failed upserting secret value in Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return task.Error(err)
	}
	return task.Respond(VaultSecretTaskResponse{})
}

func handleDelete(in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	path, err := getFullPathForDelete(in.EngineVersion, in.EngineName, in.Path)
	if err != nil {
		logrus.WithError(err).Errorf("failed deleting secret value from Vault. Url: [%s]; Path: [%s]; EngineName: [%s]; EngineVersion: [%d]", client.Address(), in.Path, in.EngineName, in.EngineVersion)
		return task.Error(err)
	}
	logrus.Infof("deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	_, err = client.Logical().Delete(path)
	if err != nil {
		logrus.WithError(err).Errorf("failed deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
		return task.Error(err)
	}
	logrus.Infof("done deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	return task.Respond(VaultSecretTaskResponse{})
}

func upsert(engineVersion uint8, engineName string, path string, key string, value string, client *vault.Client) error {
	data := map[string]any{
		"data": map[string]string{
			key: value,
		},
	}
	fullPath, err := getFullPath(engineVersion, engineName, path)
	if err != nil {
		return err
	}
	logrus.Infof("writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	_, err = client.Logical().Write(fullPath, data)
	if err != nil {
		return err
	}
	logrus.Infof("done writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	return nil
}

func fetch(engineVersion uint8, engineName string, path string, key string, client *vault.Client) (string, error) {
	logrus.Infof("fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	fullPath, err := getFullPath(engineVersion, engineName, path)
	if err != nil {
		return "", err
	}
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
	err = fmt.Errorf("could not find key [%s] in secret data", key)
	return "", err
}

func getFullPath(engineVersion uint8, engineName string, path string) (string, error) {
	switch engineVersion {
	case 1:
		return fmt.Sprintf("%s/%s", engineName, path), nil
	case 2:
		return fmt.Sprintf("%s/data/%s", engineName, path), nil
	default:
		return "", fmt.Errorf("unsupported secret engine version [%d]", engineVersion)
	}
}

func getFullPathForDelete(engineVersion uint8, secretEngineName string, path string) (string, error) {
	switch engineVersion {
	case 1:
		return fmt.Sprintf("%s/%s", secretEngineName, path), nil
	case 2:
		return fmt.Sprintf("%s/metadata/%s", secretEngineName, path), nil
	default:
		return "", fmt.Errorf("unsupported secret engine version [%d]", engineVersion)
	}
}
