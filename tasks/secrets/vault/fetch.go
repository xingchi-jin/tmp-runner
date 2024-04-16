package vault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/drone/go-task/task"
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

type vaultSecret struct {
	Config *Config `json:"config"`
	Path   string  `json:"path"`
	Key    string  `json:"key"`
}

type input struct {
	Secrets []*vaultSecret `json:"secrets"`
}

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

	secret, err := client.Logical().Read(in.Secrets[0].Path)
	if err != nil {
		return task.Error(err)
	}
	if secret == nil || secret.Data == nil {
		return task.Error(fmt.Errorf("could not find secret: %s", in.Secrets[0].Path))
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
		if k == in.Secrets[0].Key {
			return task.Respond(
				&task.Secret{
					Value: s,
				},
			)
		}
	}

	return task.Error(fmt.Errorf("could not find secret key: %s", in.Secrets[0].Key))
}
