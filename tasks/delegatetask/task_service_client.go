package delegatetask

import (
	"bytes"
	"context"

	"github.com/harness/runner/utils"
)

var client Client

func SendTask(ctx context.Context, data []byte) error {
	if client == nil {
		endpoint := ctx.Value("task_service_url").(string)
		skipVerify := ctx.Value("skip_verify").(bool)
		client = NewTaskServiceClient(endpoint, skipVerify, "")
	}
	return client.SendTask(ctx, data)
}

type Client interface {
	SendTask(ctx context.Context, data []byte) error
}

type TaskServiceClient struct {
	utils.HTTPClient
}

func NewTaskServiceClient(endpoint string, skipVerify bool, additionalCertsDir string) *TaskServiceClient {
	return &TaskServiceClient{
		HTTPClient: *utils.New(endpoint, skipVerify, additionalCertsDir),
	}
}

func (c *TaskServiceClient) SendTask(ctx context.Context, data []byte) error {
	if _, _, err := c.Do(ctx, "/api/tasks", "POST", map[string]string{"Content-Type": "application/x-kryo-v2"}, bytes.NewBuffer(data)); err != nil {
		c.Logger.Error("Sending request to task service failed with ", err)
		return err
	}
	return nil
}
