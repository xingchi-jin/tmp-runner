package vault

import (
	"context"
	"fmt"

	"github.com/harness/runner/logger"
	vault "github.com/hashicorp/vault/api"
)

func upsert(ctx context.Context, client *vault.Client, engineVersion uint8, engineName, path, key, value string) (string, error) {
	data := map[string]any{
		"data": map[string]string{
			key: value,
		},
	}
	fullPath, err := getFullPath(engineVersion, engineName, path)
	if err != nil {
		return "", err
	}
	logger.Infof(ctx, "writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	_, err = client.Logical().Write(fullPath, data)
	return fullPath, err
}

func delete(ctx context.Context, client *vault.Client, engineVersion uint8, engineName, path string) (string, error) {
	fullPath, err := getFullPathForDelete(engineVersion, engineName, path)
	if err != nil {
		return "", err
	}
	logger.Infof(ctx, "deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	_, err = client.Logical().Delete(fullPath)
	return path, err
}

func validate(client *vault.Client, engineVersion uint8, engineName, path string) (string, error) {
	fullPath, err := getFullPath(engineVersion, engineName, path)
	if err != nil {
		return "", err
	}
	if _, err := fetch(client, fullPath); err != nil {
		return "", err
	}
	return fullPath, nil
}

func fetch(client *vault.Client, path string) (*vault.Secret, error) {
	secret, err := client.Logical().Read(path)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("could not find secret: %s", path)
	}

	v := secret.Data["data"]
	if data, ok := v.(map[string]interface{}); ok {
		secret.Data = data
	}
	return secret, nil
}
