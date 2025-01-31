package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/common"
	"github.com/harness/runner/logger"
	vault "github.com/hashicorp/vault/api"
)

func Handler(ctx context.Context, req *task.Request) task.Response {
	in := new(VaultSecretTaskRequest)

	// decode the task input.
	err := json.Unmarshal(req.Task.Data, in)
	if err != nil {
		logger.Errorf(ctx, "failed to unmarshal task input, %s", err)
		return task.Respond(NewErrorResponse(err, "Failed to decode task input", http.StatusBadRequest))
	}

	client, err := New(in.Config)
	if err != nil {
		logger.Errorf(ctx, "failed to create vault client, %s", err)
		return task.Respond(NewErrorResponse(err, "Failed to create Vault Client", http.StatusInternalServerError))
	}

	switch action := in.Action; action {
	case "CONNECT":
		return handleConnect(ctx, in, client)
	case "UPSERT":
		return handleUpsert(ctx, in, client)
	case "DELETE":
		return handleDelete(ctx, in, client)
	case "VALIDATE":
		return handleValidateRef(ctx, in, client)
	default:
		logger.Error(ctx, fmt.Errorf("unsupported secret task action: %s", action))
		return task.Respond(NewErrorResponse(fmt.Errorf("invalid action"), fmt.Sprintf("The specified action %s is not supported", action), http.StatusBadRequest))
	}
}

// FetchHandler returns a task handler that fetches a secret from vault.
func FetchHandler(ctx context.Context, req *task.Request) task.Response {
	in := new(VaultSecretFetchRequest)

	// decode the task input.
	if err := json.Unmarshal(req.Task.Data, in); err != nil {
		return task.Error(err)
	}

	client, err := New(in.Secrets[0].Config)
	if err != nil {
		return task.Error(err)
	}

	return handleFetch(ctx, in, client)
}

func handleConnect(ctx context.Context, in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	var (
		err        error = nil
		secret     *vault.Secret
		secretAuth *vault.SecretAuth
	)

	if in.Config.AppRoleId != "" && in.Config.AppSecretId != "" {
		secretAuth, err = appRoleLogin(client, in.Config.AppRoleId, in.Config.AppSecretId)
		if err == nil {
			logger.Infof(ctx, "sucessfully authenticated connection to Vault using AppRole. Url: [%s];", client.Address())
			return task.Respond(VaultConnectionResponse{
				AppRoleAuth: createAppRoleTokenObject(secretAuth),
			})
		}
	} else if in.Config.Token != "" {
		secret, err = tokenLogin(client, in.Config.Token)
		if err == nil {
			logger.Infof(ctx, "sucessfully authenticated connection to Vault using Token. Url: [%s];", client.Address())
			return task.Respond(VaultConnectionResponse{
				TokenAuth: createTokenAuthObject(secret),
			})
		}
	}

	// If no error, that means neither token nor appRole was provided
	if err == nil {
		err = fmt.Errorf("neither of token or appRole provided for authentication")
	}
	logger.WithError(ctx, err).Errorf("failed validating connection to Vault. Url: [%s];", client.Address())
	return task.Respond(VaultConnectionResponse{
		Error: err,
	})
}

func handleFetch(ctx context.Context, in *VaultSecretFetchRequest, client *vault.Client) task.Response {
	secret, err := fetch(client, in.Secrets[0].Path)
	if err != nil {
		logger.WithError(ctx, err).Errorf("failed fetching secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Secrets[0].Path)
		return task.Error(err)
	}

	decodedSecret, err := getSecretValue(in.Secrets[0].Key, in.Secrets[0].Base64, secret.Data)
	if err != nil {
		logger.WithError(ctx, err).Errorf("failed decoding secret value. Url: [%s]; Path: [%s]", client.Address(), in.Secrets[0].Path)
		return task.Error(err)
	}

	logger.Infof(ctx, "sucessfully fetched secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Secrets[0].Path)
	return task.Respond(
		&common.Secret{
			Value: decodedSecret,
		},
	)
}

func handleUpsert(ctx context.Context, in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	path, err := upsert(ctx, client, in.EngineVersion, in.EngineName, in.Path, in.Key, in.Value)
	if err != nil {
		logger.WithError(ctx, err).Errorf("failed upserting secret value in Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return task.Respond(VaultSecretOperationResponse{
			Name:    in.Key,
			Message: "Failed upserting secret value in Vault",
			Error: &Error{
				Message: "Failed upserting secret value in Vault",
				Reason:  err.Error(),
			},
			OperationStatus: OperationStatusFailure,
		})
	}
	logger.Infof(ctx, "done writing secret value to Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	return task.Respond(VaultSecretOperationResponse{
		Name:            in.Path,
		Message:         "Secret upserted to vault",
		OperationStatus: OperationStatusSuccess,
	})
}

func handleDelete(ctx context.Context, in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	path, err := delete(ctx, client, in.EngineVersion, in.EngineName, in.Path)
	if err != nil {
		logger.WithError(ctx, err).Errorf("failed deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return task.Respond(VaultSecretOperationResponse{
			Name:    in.Key,
			Message: "Failed deleting secret value in Vault",
			Error: &Error{
				Message: "Failed deleting secret value in Vault",
				Reason:  err.Error(),
			},
			OperationStatus: OperationStatusFailure,
		})
	}
	logger.Infof(ctx, "done deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), path)
	return task.Respond(VaultSecretOperationResponse{
		Name:            in.Key,
		Message:         "Secret deleted from vault",
		OperationStatus: OperationStatusSuccess,
	})
}

func handleValidateRef(ctx context.Context, in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	path, err := validate(client, in.EngineVersion, in.EngineName, in.Path)
	if err != nil {
		logger.WithError(ctx, err).Errorf("failed finding secret secret reference in Vault. Url: [%s]; Path: [%s]", client.Address(), in.Path)
		return task.Respond(ValidationResponse{
			IsValid: false,
			Error: &Error{
				Message: "Failed finding secret reference in Vault",
				Reason:  err.Error(),
			},
		})
	}

	logger.Infof(ctx, "secret reference found in vault. Url: [%s]; Path: [%s]", client.Address(), path)
	return task.Respond(ValidationResponse{
		IsValid: true,
	})
}
