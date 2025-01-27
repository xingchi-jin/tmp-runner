package vault

import (
	"fmt"

	vault "github.com/hashicorp/vault/api"
)

// Vault API Url
const (
	tokenAuthUrl   = "/auth/token/lookup"
	appRoleAuthUrl = "/auth/approle/login"
)

// Config provides the vault configuration.
type Config struct {
	Address     string `json:"address"`
	Token       string `json:"token"`
	Namespace   string `json:"namespace"`
	AppRoleId   string `json:"app_role_id"`
	AppSecretId string `json:"app_role_secret"`
}

// New returns a new vault client.
func New(in *Config) (*vault.Client, error) {
	config := vault.DefaultConfig()
	if in == nil {
		return vault.NewClient(config)

	}
	if in.Address != "" {
		config.Address = in.Address
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, err
	}

	// Use AppRole based login if provided
	if secret, err := appRoleLogin(client, in.AppRoleId, in.AppSecretId); err == nil {
		in.Token = secret.ClientToken
	}

	if in.Token != "" {
		client.SetToken(in.Token)
	}

	if in.Namespace != "" {
		client.SetNamespace(in.Namespace)
	}
	return client, nil
}

// appRoleLogin performs login using appRoleId and appRoleSecret and returns the auth response data
func appRoleLogin(client *vault.Client, appRoleId, appRoleSecret string) (*vault.SecretAuth, error) {
	if appRoleId == "" || appRoleSecret == "" {
		return nil, fmt.Errorf("appRoleId and/or appRoleSecret not provided")
	}

	secret, err := client.Logical().Write(appRoleAuthUrl, map[string]interface{}{
		"role_id":   appRoleId,
		"secret_id": appRoleSecret,
	})
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Auth == nil {
		return nil, fmt.Errorf("received empty auth token response from AppRole login")
	}
	return secret.Auth, nil
}

// tokenLogin performs lookup using the provided token and returns the auth response data
func tokenLogin(client *vault.Client, token string) (*vault.Secret, error) {
	if token == "" {
		return nil, fmt.Errorf("token not provided")

	}
	secret, err := client.Logical().Write(tokenAuthUrl, map[string]interface{}{
		"token": token,
	})
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("received empty secret response from token lookup")
	}
	return secret, nil
}
