package vault

import (
	"encoding/base64"
	"fmt"

	"github.com/harness/runner/utils"
	vault "github.com/hashicorp/vault/api"
)

// getFullPath returns the fullPath based on the engineVersion
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

// getFullPathForDelete returns the fullPath based on the engineVersion for delete operation
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

// getSecretValue extracts the secret value using the provided key from data
// and returns decoded secret
func getSecretValue(key string, base64 bool, data map[string]interface{}) (string, error) {
	for k, v := range data {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if k == key {
			return parseSecretString(s, base64)
		}
	}
	return "", fmt.Errorf("could not find secret key: %s", key)
}

// parseSecretString will decode the secret string
func parseSecretString(s string, decode bool) (string, error) {
	if !decode {
		decoded, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return "", fmt.Errorf("error occurred when decoding base64 secret. %w", err)
		}
		s = string(decoded)
	}
	return s, nil
}

func createAppRoleTokenObject(secretAuth *vault.SecretAuth) AppRoleToken {
	return AppRoleToken{
		ClientToken:   secretAuth.ClientToken,
		Accessor:      secretAuth.Accessor,
		Policies:      secretAuth.Policies,
		LeaseDuration: secretAuth.LeaseDuration,
		Renewable:     secretAuth.Renewable,
	}
}

func createTokenAuthObject(secret *vault.Secret) AuthToken {
	return AuthToken{
		Name:       utils.SafeStringAssertion(secret.Data, "display_name", ""),
		ExpiryTime: utils.SafeStringAssertion(secret.Data, "expire_time", ""),
		Renewable:  utils.SafeBoolAssertion(secret.Data, "renewable", false),
	}
}
