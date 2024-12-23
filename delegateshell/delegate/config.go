// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/drone-runners/drone-runner-aws/command/config"
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

// Config Sample config
type Config struct {
	Debug               bool `envconfig:"DEBUG"`
	Trace               bool `envconfig:"TRACE"`
	EnableRemoteLogging bool `envconfig:"ENABLE_REMOTE_LOGGING" default:"false"`

	Delegate struct {
		ID              string // This is populated after a successful registration call to the manager
		AccountID       string `envconfig:"ACCOUNT_ID"`
		Token           string `envconfig:"DELEGATE_TOKEN"`
		Tags            string `envconfig:"DELEGATE_TAGS" split_words:"true"`
		ManagerEndpoint string `envconfig:"MANAGER_HOST_AND_PORT"`
		Name            string `envconfig:"DELEGATE_NAME"`

		ParallelWorkers       int `envconfig:"PARALLEL_WORKERS" default:"100"`
		PollIntervalMilliSecs int `envconfig:"POLL_INTERVAL_MILLISECS" default:"3000"`

		TaskServiceURL string     `envconfig:"TASK_SERVICE_URL" default:"http://localhost:3461"`
		Type           RunnerType `envconfig:"DELEGATE_TYPE"`
		RunnerType     RunnerType `envconfig:"RUNNER_TYPE"`
		MaxStages      *int       `envconfig:"MAX_STAGES"`
	}

	Server struct {
		Bind              string `envconfig:"HTTPS_BIND" default:":3000"`
		CertFile          string `envconfig:"SERVER_CERT_FILE" default:"/tmp/certs/server-cert.pem"` // Server certificate PEM file
		KeyFile           string `envconfig:"SERVER_KEY_FILE" default:"/tmp/certs/server-key.pem"`   // Server key PEM file
		CACertFile        string `envconfig:"CLIENT_CERT_FILE" default:"/tmp/certs/ca-cert.pem"`     // CA certificate file
		SkipPrepareServer bool   `envconfig:"SKIP_PREPARE_SERVER" default:"false"`                   // skip prepare server, install docker / git
		Insecure          bool   `envconfig:"SERVER_INSECURE" default:"true"`                        // run in insecure mode
	}

	// Config needed to be able to run VM builds on the runners
	VM struct {
		Database struct {
			Driver     string `envconfig:"VM_DATABASE_DRIVER" default:"postgres"`
			Datasource string `envconfig:"VM_DATABASE_DATASOURCE" default:"port=5431 user=admin password=password dbname=dlite sslmode=disable"`
		}

		BinaryURI struct {
			LiteEngine    string `envconfig:"VM_BINARY_URI_LITE_ENGINE" default:"https://github.com/harness/lite-engine/releases/download/v0.5.88/"`
			Plugin        string `envconfig:"VM_BINARY_URI_PLUGIN" default:"https://github.com/drone/plugin/releases/download/v0.3.8-beta"`
			AutoInjection string `envconfig:"VM_BINARY_AUTO_INJECTION" default:"https://app.harness.io/storage/harness-download/harness-ti/auto-injection/1.0.3"`
			SplitTests    string `envconfig:"VM_BINARY_URI_SPLIT_TESTS" default:"https://app.harness.io/storage/harness-download/harness-ti/split_tests"`
		}

		Pool struct {
			File              string              `envconfig:"VM_POOL_FILE"`
			MapByAccountID    PoolMapperByAccount `envconfig:"VM_POOL_MAP_BY_ACCOUNT_ID"`
			BusyMaxAge        int64               `envconfig:"VM_POOL_BUSY_MAX_AGE" default:"24"`
			FreeMaxAge        int64               `envconfig:"VM_POOL_FREE_MAX_AGE" default:"720"`
			PurgerTimeMinutes int64               `envconfig:"VM_POOL_PURGER_TIME_MINUTES" default:"30"`
		}

		Password struct {
			Tart      string `envconfig:"VM_PASSWORD_TART"`
			AnkaToken string `envconfig:"VM_PASSWORD_ANKA_TOKEN"`
		}
	}

	Metrics struct {
		Provider string `envconfig:"METRICS_PROVIDER" default:"prometheus"`
		Endpoint string `envconfig:"METRICS_ENDPOINT" default:"/metrics"`
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

type TaskContext struct {
	DelegateTaskServiceURL string  // URL of Delegate Task Service
	DelegateId             *string // Delegate id gets set after a successful runner registration call.
	DelegateName           string
	SkipVerify             bool       // Skip SSL verification if the task is conducting https connection.
	RunnerType             RunnerType // The type of the runner
	ManagerEndpoint        string
	AccountID              string // Account ID associated with the runner
	Token                  string

	PoolMapperByAccount map[string]map[string]string
}

type CapacityConfig struct {
	MaxStages *int
}

// Iterates over all the entries and converts it to a simple type
func (pma *PoolMapperByAccount) Convert() map[string]map[string]string {
	m := map[string]map[string]string{}
	for k, v := range *pma {
		m[k] = map[string]string(v)
	}
	return m
}

type PoolMap map[string]string
type PoolMapperByAccount map[string]PoolMap

func (pma *PoolMapperByAccount) Decode(value string) error {
	m := map[string]PoolMap{}
	pairs := strings.Split(value, ";")
	for _, pair := range pairs {
		p := PoolMap{}
		kvpair := strings.Split(pair, "=")
		if len(kvpair) != 2 { //nolint:gomnd
			return fmt.Errorf("invalid map item: %q", pair)
		}
		err := json.Unmarshal([]byte(kvpair[1]), &p)
		if err != nil {
			return fmt.Errorf("invalid map json: %w", err)
		}
		m[kvpair[0]] = p
	}
	*pma = PoolMapperByAccount(m)
	return nil
}

func FromEnviron() (*Config, error) {
	var config Config
	err := envconfig.Process("", &config)
	if err != nil {
		return &config, err
	}

	return &config, nil
}

// Upsert updates any fields in the config which are set after reading from
// the environment.
func (c *Config) UpsertDelegateID(delegateID string) {
	if c.Delegate.ID == "" {
		c.Delegate.ID = delegateID
	}
}

// GetTags returns the list of tags for the runner.
// If a pool file is specified, it parses the tags from the pool file and appends them to the tags.
func (c *Config) GetTags() []string {
	tags := make([]string, 0)
	for _, s := range strings.Split(pickNonEmpty(c.Selectors, c.Delegate.Tags), ",") {
		tags = append(tags, strings.TrimSpace(s))
	}

	// append tags present in the pool file
	if c.VM.Pool.File != "" {
		configPool, err := config.ParseFile(c.VM.Pool.File)
		if err == nil {
			tags = append(tags, parseTags(configPool)...)
		}
	}
	return tags
}

func parseTags(pf *config.PoolFile) []string {
	tags := []string{}
	for i := range pf.Instances {
		tags = append(tags, pf.Instances[i].Name)
	}
	return tags
}

func (c *Config) GetName() string {
	return pickNonEmpty(c.RunnerName, c.Delegate.Name)
}

// Token copied from Harness Saas UI is base64 encoded. However, since kubernetes secret is used to create the token
// with token value put in 'data' field of secret yaml as plain text, the token passed to delegate agent is already
// decoded. For Docker delegates, token passes to delegate agent is still not decoded. This function is to provide
// compatibility to both use cases.
func getBase64DecodedTokenString(token string) string {
	// Step 1: Check if the token is base64 encoded
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return token // Not base64, return original token
	}

	// Step 2: Decode the token, check if the decoded result is a hexadecimal string
	decodedStr := strings.TrimSpace(string(decoded))
	if isHexadecimalString(decodedStr) {
		return decodedStr
	}
	return token
}

// A helper function to check if a string is a 32-character hexadecimal string
func isHexadecimalString(decodedToken string) bool {
	match, _ := regexp.MatchString("^[0-9A-Fa-f]{32}$", decodedToken)
	return match
}

func (c *Config) GetHarnessUrl() string {
	return pickNonEmpty(c.HarnessUrl, c.Delegate.ManagerEndpoint)
}

func (c *Config) GetToken() string {
	secret := pickNonEmpty(c.Token, c.Delegate.Token)
	return getBase64DecodedTokenString(secret)
}

func pickNonEmpty(str1, str2 string) string {
	if str1 != "" {
		return str1
	}
	return str2
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

func (c *Config) GetRunnerType() RunnerType {
	if c.Delegate.RunnerType != "" {
		return c.Delegate.RunnerType
	}
	return c.Delegate.Type
}

func (c *Config) GetCapacityConfig() CapacityConfig {
	return CapacityConfig{MaxStages: c.Delegate.MaxStages}
}
