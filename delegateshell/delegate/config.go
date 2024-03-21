// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegate

import (
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Sample config
type Config struct {
	Debug bool `envconfig:"DEBUG"`
	Trace bool `envconfig:"TRACE"`

	Delegate struct {
		AccountID       string `envconfig:"ACCOUNT_ID"`
		DelegateToken   string `envconfig:"DELEGATE_TOKEN"`
		Tags            string `envconfig:"DELEGATE_TAGS"`
		ManagerEndpoint string `envconfig:"MANAGER_HOST_AND_PORT"`
		Name            string `envconfig:"DELEGATE_NAME"`
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
