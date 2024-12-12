package vault

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/common"
	"github.com/hashicorp/vault/api"
)

// Sample handler that reads a secret from vault.
//
// Sample json input:
//
// {
//     "task": {
//         "id": "67c0938c-9348-4c5e-8624-28218984e09g",
//         "type": "secret/vault/fetch",
//         "data": {
//             "secrets": [
//               {
//                   "config": {
//                        "address": "http://localhost:8200",
//                        "token": "root"
//                   },
//                   "path": "secret/data/aws_secret",
//                   "key": "aws_secret"
// 	   			 }
//             ]
//         }
//     }
// }

// FetchHandler returns a task handler that fetches a secret from vault.
func FetchHandler(ctx context.Context, req *task.Request) task.Response {
	in := new(input)

	// decode the task input.
	err := json.Unmarshal(req.Task.Data, in)
	if err != nil {
		return task.Error(err)
	}

	client, err := New(in.Secrets[0].Config)
	if err != nil {
		return task.Error(err)
	}

	secret, err := fetchSecret(client, in)
	if err != nil {
		return task.Error(err)
	}

	decodedSecret, err := getSecretKeyAndValue(in, secret)

	if err != nil {
		return task.Error(err)
	}
	return task.Respond(
		&common.Secret{
			Value: decodedSecret,
		},
	)
}

func fetchSecret(client *api.Client, in *input) (*api.Secret, error) {
	secret, err := client.Logical().Read(in.Secrets[0].Path)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("could not find secret: %s", in.Secrets[0].Path)
	}

	v := secret.Data["data"]
	if data, ok := v.(map[string]interface{}); ok {
		secret.Data = data
	}
	return secret, nil
}

func getSecretKeyAndValue(in *input, secret *api.Secret) (string, error) {
	for k, v := range secret.Data {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if k == in.Secrets[0].Key {
			return parse(s, in.Secrets[0].Base64)
		}
	}
	return "", fmt.Errorf("could not find secret key: %s", in.Secrets[0].Key)
}

func parse(s string, decode bool) (string, error) {
	if decode {
		decoded, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return "", fmt.Errorf("error occurred when decoding base64 secret. %w", err)
		}
		s = string(decoded)
	}
	return s, nil
}
