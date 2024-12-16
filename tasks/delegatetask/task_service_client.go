package delegatetask

import (
	"bytes"
	"context"
	"sync"

	"github.com/harness/runner/logger"

	"github.com/harness/runner/utils"
)

var once sync.Once
var client Client

func SendTask(ctx context.Context, data []byte, url string, skipVerify bool) error {
	once.Do(func() {
		client = NewTaskServiceClient(url, skipVerify, "")
	})
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
		logger.Error(ctx, "Sending request to task service failed with ", err)
		return err
	}
	return nil
}
