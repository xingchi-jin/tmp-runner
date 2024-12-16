package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/harness/runner/logger"

	"github.com/drone/go-task/task"
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
	case "UPSERT":
		return handleUpsert(ctx, in, client)
	case "DELETE":
		return handleDelete(ctx, in, client)
	default:
		logger.Error(ctx, fmt.Errorf("unsupported secret task action: %s", action))
		return task.Respond(NewErrorResponse(fmt.Errorf("invalid action"), fmt.Sprintf("The specified action %s is not supported", action), http.StatusBadRequest))
	}
}

func handleUpsert(ctx context.Context, in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	path, err := upsert(ctx, in.EngineVersion, in.EngineName, in.Path, in.Key, in.Value, client)
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
		Name:            in.Key,
		Message:         "Secret upserted to vault",
		OperationStatus: OperationStatusSuccess,
	})
}

func handleDelete(ctx context.Context, in *VaultSecretTaskRequest, client *vault.Client) task.Response {
	path, err := delete(ctx, in.EngineVersion, in.EngineName, in.Path, client)
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

func upsert(ctx context.Context, engineVersion uint8, engineName string, path string, key string, value string, client *vault.Client) (string, error) {
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

func delete(ctx context.Context, engineVersion uint8, engineName, path string, client *vault.Client) (string, error) {
	fullPath, err := getFullPathForDelete(engineVersion, engineName, path)
	if err != nil {
		return "", err
	}
	logger.Infof(ctx, "deleting secret value from Vault. Url: [%s]; Path: [%s]", client.Address(), fullPath)
	_, err = client.Logical().Delete(fullPath)
	return path, err
}

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
