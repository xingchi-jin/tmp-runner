package delegatetask

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/drone/go-task/task"
	"github.com/sirupsen/logrus"
)

func DelegateTaskHandler(ctx context.Context, req *task.Request) task.Response {
	var delegateTask DelegateTaskRequest
	err := json.Unmarshal(req.Task.Data, &delegateTask)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
		return task.Error(err)
	}
	// invoke API call to pass the data to delegate-task-service
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(delegateTask.Base64Data)))
	n, err := base64.StdEncoding.Decode(dst, delegateTask.Base64Data)
	if err != nil {
		logrus.WithError(err).Error("Decode delegate task package error with base64")
		return nil
	}
	dst = dst[:n]

	if err = SendTask(ctx, dst); err != nil {
		logrus.WithError(err).Error("Send request to delegate task service failed")
		return task.Error(err)
	}
	// The response of delegate task is sent by the delegate task service
	return nil
}

type DelegateTaskRequest struct {
	Base64Data []byte `json:"taskPackage"`
}

