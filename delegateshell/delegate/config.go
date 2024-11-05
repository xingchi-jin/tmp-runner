// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegate

import (
	"fmt"
	"os"
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
		AccountID       string `envconfig:"ACCOUNT_ID"`
		DelegateToken   string `envconfig:"DELEGATE_TOKEN"`
		Tags            string `envconfig:"DELEGATE_TAGS" split_words:"true"`
		ManagerEndpoint string `envconfig:"MANAGER_HOST_AND_PORT"`
		Name            string `envconfig:"DELEGATE_NAME"`

		TaskStatusV2           bool       `envconfig:"DELEGATE_TASK_STATUS_V2" default:"true"`
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

	// Runner's installation configs
	// Certain congigs will deprecate the old ones in order to provide better environment variable names.
	// eg. TOKEN replaces DELEGATE_TOKEN, TAGS replaces DELEGATE_TAGS, URL replaces MANAGER_HOST_AND_PORT, NAME replaces DELEGATE_NAME
	// Note: If both new and old name are configured, the new name takes precedence.
	//
	// Note: Putting configs inside a nested struct will cause envconfig failing to load the value, because envconfig
	// will prefix the intended env name with the name of the struct. For example, if we define "Token" inside "Delegate" struct with flag
	// 'envconfig:"TOKEN"', it will be loaded by environment variable "DELEGATE_TOKEN"

	Token      string `envconfig:"TOKEN"`
	Selectors  string `envconfig:"TAGS"`
	RunnerName string `envconfig:"NAME"`
	HarnessUrl string `envconfig:"URL"`
}

func FromEnviron() (Config, error) {
	var config Config
	for _, e := range os.Environ() {
		fmt.Println(e)
	}
	err := envconfig.Process("", &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func (c *Config) GetTags() []string {
	tags := make([]string, 0)
	for _, s := range strings.Split(pickNonEmpty(c.Selectors, c.Delegate.Tags), ",") {
		tags = append(tags, strings.TrimSpace(s))
	}
	return tags
}

func (c *Config) GetName() string {
	return pickNonEmpty(c.RunnerName, c.Delegate.Name)
}

func (c *Config) GetHarnessUrl() string {
	return pickNonEmpty(c.HarnessUrl, c.Delegate.ManagerEndpoint)
}

func (c *Config) GetToken() string {
	return pickNonEmpty(c.Token, c.Delegate.DelegateToken)
}

func pickNonEmpty(str1, str2 string) string {
	if str1 != "" {
		return str1
	}
	return str2
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
