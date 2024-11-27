// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metricsinjection

import (
	"strings"

	"github.com/google/wire"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/metrics"
	metricshandler "github.com/harness/runner/metrics/handler"
	"github.com/harness/runner/metrics/providers/prometheus"
)

var WireSet = wire.NewSet(
	ProvideMetricsHandler,
	ProvideMetricsClient,
)

func ProvideMetricsHandler(config *delegate.Config) *metricshandler.MetricsHandler {
	return metricshandler.NewMetricsHandler(config.Metrics.Provider, config.Metrics.Endpoint)
}

func ProvideMetricsClient(config *delegate.Config) metrics.Metrics {
	switch strings.ToLower(config.Metrics.Provider) {
	case "prometheus":
		return prometheus.NewPrometheusMetrics()
	default:
		return prometheus.NewPrometheusMetrics()
	}
}
