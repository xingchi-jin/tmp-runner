// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metricshandler

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsHandler struct {
	Endpoint string `json:"metrics_endpoint,omitempty"`
	Provider string `json:"metrics_provider"`
}

func NewMetricsHandler(provider, endpoint string) *MetricsHandler {
	return &MetricsHandler{
		Endpoint: endpoint,
		Provider: provider,
	}
}

func (ms *MetricsHandler) Handle() {
	endpoint := ms.Endpoint
	switch ms.Provider {
	case "prometheus":
		http.Handle(endpoint, promhttp.Handler())
	default:
		http.Handle(endpoint, promhttp.Handler())
	}
}
