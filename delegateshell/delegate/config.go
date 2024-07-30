// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegate

import (
	"strings"
	"sync"

	"github.com/kelseyhightower/envconfig"
)

const (
	RunnerTypeShell       RunnerType = "SHELL"
	RunnerTypeShellScript RunnerType = "SHELL_SCRIPT"
	RunnerTypeDocker      RunnerType = "DOCKER"
	RunnerTypeK8s         RunnerType = "KUBERNETES"
	RunnerTypeCEK8s       RunnerType = "CE_KUBERNETES"
	RunnerTypeHelm        RunnerType = "HELM_DELEGATE"
	RunnerTypeECS         RunnerType = "ECS"
)

type RunnerType string

// Sample config
type Config struct {
	Debug bool `envconfig:"DEBUG"`
	Trace bool `envconfig:"TRACE"`

	Delegate struct {
		AccountID              string     `envconfig:"ACCOUNT_ID"`
		DelegateToken          string     `envconfig:"DELEGATE_TOKEN"`
		Tags                   string     `envconfig:"DELEGATE_TAGS"`
		ManagerEndpoint        string     `envconfig:"MANAGER_HOST_AND_PORT"`
		Name                   string     `envconfig:"DELEGATE_NAME"`
		TaskStatusV2           bool       `envconfig:"DELEGATE_TASK_STATUS_V2"`
		DelegateTaskServiceURL string     `envconfig:"TASK_SERVICE_URL" default:"http://localhost:3461"`
		DelegateType           RunnerType `envconfig:"DELEGATE_TYPE"`
		RunnerType             RunnerType `envconfig:"RUNNER_TYPE"`
	}

	Server struct {
		Bind              string `envconfig:"HTTPS_BIND" default:":3000"`
		CertFile          string `envconfig:"SERVER_CERT_FILE" default:"/tmp/certs/server-cert.pem"` // Server certificate PEM file
		KeyFile           string `envconfig:"SERVER_KEY_FILE" default:"/tmp/certs/server-key.pem"`   // Server key PEM file
		CACertFile        string `envconfig:"CLIENT_CERT_FILE" default:"/tmp/certs/ca-cert.pem"`     // CA certificate file
		SkipPrepareServer bool   `envconfig:"SKIP_PREPARE_SERVER" default:"false"`                   // skip prepare server, install docker / git
		Insecure          bool   `envconfig:"SERVER_INSECURE" default:"true"`                        // run in insecure mode
	}
}

func FromEnviron() (Config, error) {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func (c *Config) GetTags() []string {
	tags := make([]string, 0)
	for _, s := range strings.Split(c.Delegate.Tags, ",") {
		tags = append(tags, strings.TrimSpace(s))
	}
	return tags
}

// Configurations that will pass to task handlers at runtime
type TaskContext struct {
	DelegateTaskServiceURL string     // URL of Delegate Task Service
	DelegateId             string     // Delegate id abtained after a successful runner registration call.
	SkipVerify             bool       // Skip SSL verification if the task is conducting https connection.
	RunnerType             RunnerType // The type of the runner
}

var once sync.Once
var taskContext *TaskContext

func GetTaskContext(config *Config, delegateId string) *TaskContext {
	once.Do(func() {
		taskContext = &TaskContext{
			DelegateId:             delegateId,
			DelegateTaskServiceURL: config.Delegate.DelegateTaskServiceURL,
			SkipVerify:             config.Server.Insecure,
			RunnerType:             getRunnerType(config),
		}
	})
	return taskContext
}

func IsK8sRunner(runnerType RunnerType) bool {
	switch runnerType {
	case RunnerTypeK8s:
		return true
	case RunnerTypeCEK8s:
		return true
	case RunnerTypeHelm:
		return true
	default:
		return false
	}
}

func getRunnerType(config *Config) RunnerType {
	if config.Delegate.RunnerType != "" {
		return config.Delegate.RunnerType
	}
	return config.Delegate.DelegateType
}
