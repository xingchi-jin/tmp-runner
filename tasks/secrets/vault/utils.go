package vault

import (
	"fmt"

	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type SecretTaskResponse struct{}

func Upsert(engineName string, path string, key string, value string, oldPath string, deleteOldPath bool, client *vault.Client) error {
	data := map[string]any{
		"data": map[string]string{
			key: value,
		},
	}
	fullPath := GetFullPath(engineName, path)
	logrus.Infof("Writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	_, err := client.Logical().Write(fullPath, data)
	if err != nil {
		logrus.WithError(err).Errorf("Failed writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
		return err
	}
	logrus.Infof("Done writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	if deleteOldPath {
		fullOldPath := GetFullPathForDelete(engineName, oldPath)
		logrus.Infof("Deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)
		_, err = client.Logical().Delete(fullOldPath)
		if err != nil {
			logrus.WithError(err).Errorf("Failed deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)
		}
		logrus.Infof("Done deleting previous secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullOldPath)

	}
	return nil
}

func Fetch(engineName string, path string, key string, client *vault.Client) (string, error) {
	logrus.Infof("Fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	fullPath := GetFullPath(engineName, path)
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

func GetFullPath(engineName string, path string) string {
	return fmt.Sprintf("%s/data/%s", engineName, path)
}

func GetFullPathForDelete(secretEngineName string, path string) string {
	return fmt.Sprintf("%s/metadata/%s", secretEngineName, path)
}
