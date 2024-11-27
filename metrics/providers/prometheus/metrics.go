// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prometheus

import (
	"github.com/harness/runner/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetrics struct {
	TaskCompletedCount                *prometheus.CounterVec
	TaskFailedCount                   *prometheus.CounterVec
	TaskRunningCount                  *prometheus.GaugeVec
	TaskTimeoutCount                  *prometheus.CounterVec
	TaskExecutionTime                 *prometheus.GaugeVec
	HeartbeatFailureCount             *prometheus.CounterVec
	ErrorCount                        *prometheus.CounterVec
	CPUPercentile                     *prometheus.HistogramVec
	MemoryPercentile                  *prometheus.HistogramVec
	PoolFallbackCount                 *prometheus.CounterVec
	WaitDurationCount                 *prometheus.HistogramVec
	TaskRejectedCount                 *prometheus.CounterVec
	ResourceConsumptionAboveThreshold *prometheus.GaugeVec
}

// TaskCompletedCount provides metrics for total number of pipeline executions (failed + successful)
func TaskCompletedCount() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metrics.MetricNamePrefix + "_task_completed_total",
			Help: "Total number of completed pipeline executions (failed + successful)",
		},
		[]string{"account_id", "task_type", "runner_name"},
	)
}

// TaskFailedCount provides metrics for total failed pipeline executions
func TaskFailedCount() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metrics.MetricNamePrefix + "_task_failed_total",
			Help: "Total number of pipeline executions which failed due to errors",
		},
		[]string{"account_id", "task_type", "runner_name"},
	)
}

// TaskRunningCount provides metrics for number of tasks currently running
func TaskRunningCount() *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metrics.MetricNamePrefix + "_tasks_currently_executing",
			Help: "Total number of running executions",
		},
		[]string{"account_id", "task_type", "runner_name"},
	)
}

// TaskTimeoutCount provides metrics for number of tasks that got timed out
func TaskTimeoutCount() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metrics.MetricNamePrefix + "_task_timeout_total",
			Help: "Total number of tasks timed out",
		},
		[]string{"account_id", "task_type", "runner_name"},
	)
}

// TaskExecutionTime provides metrics for the duration of task executions
func TaskExecutionTime() *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metrics.MetricNamePrefix + "_task_execution_time",
			Help: "Task execution time per task",
		},
		[]string{"account_id", "task_type", "task_id", "runner_name"},
	)
}

func HeartbeatFailureCount() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metrics.MetricNamePrefix + "_runner_heartbeat_failure_count",
			Help: "Number of heartbeat call failures for runner",
		},
		[]string{"account_id", "runner_name"},
	)
}

// ErrorCount provides metrics for total errors in the system
func ErrorCount() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metrics.MetricNamePrefix + "_runner_error_count",
			Help: "Total number of system errors",
		},
		[]string{"account_id", "runner_name"},
	)
}

// CPUPercentile provides information about the max CPU usage in the pipeline run
func CPUPercentile() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    metrics.MetricNamePrefix + "task_max_cpu_usage_percent",
			Help:    "Max CPU usage in the pipeline",
			Buckets: []float64{30, 50, 70, 90},
		},
		[]string{"account_id", "runner_name"},
	)
}

// MemoryPercentile provides information about the max RAM usage in the pipeline run
func MemoryPercentile() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    metrics.MetricNamePrefix + "task_max_mem_usage_percent",
			Help:    "Max memory usage in the pipeline",
			Buckets: []float64{30, 50, 70, 90},
		},
		[]string{"account_id", "runner_name"},
	)
}

// PoolFallbackCount provides metrics for number of fallbacks while finding a valid pool
func PoolFallbackCount() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: metrics.MetricNamePrefix + "_ci_pipeline_pool_fallbacks",
			Help: "Total number of fallbacks triggered on the pool",
		},
		[]string{"account_id", "runner_name", "success"}, // success is true/false depending on whether fallback happened successfully
	)
}

// WaitDurationCount provides metrics for amount of time needed to wait to setup a machine
func WaitDurationCount() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    metrics.MetricNamePrefix + "_ci_runner_wait_duration_seconds",
			Help:    "Waiting time needed to successfully allocate a machine",
			Buckets: []float64{5, 15, 30, 60, 300, 600},
		},
		[]string{"account_id", "runner_name", "is_fallback"},
	)
}

// ResourceConsumptionAboveThreshold provides metrics for runner whose resource consumption is above the threshold provided
func ResourceConsumptionAboveThreshold() *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metrics.MetricNamePrefix + "_resource_consumption_above_threshold",
			Help: "If the runner resource consumption is above the threshold",
		},
		[]string{"account_id", "runner_name"},
	)
}

// registerMetrics sets up the metrics client and returns the Metrics object.
func registerMetrics() metrics.Metrics {
	taskCompletedCount := TaskCompletedCount()
	taskFailedCount := TaskFailedCount()
	taskRunningCount := TaskRunningCount()
	taskTimeoutCount := TaskTimeoutCount()
	taskExecutionTime := TaskExecutionTime()
	heartbeatFailureCount := HeartbeatFailureCount()
	errorCount := ErrorCount()
	cpuPercentile := CPUPercentile()
	memoryPercentile := MemoryPercentile()
	poolFallbackCount := PoolFallbackCount()
	waitDurationCount := WaitDurationCount()
	resourceConsumptionAboveThreshold := ResourceConsumptionAboveThreshold()

	prometheus.MustRegister(taskCompletedCount, taskFailedCount, taskRunningCount, taskTimeoutCount, taskExecutionTime, heartbeatFailureCount, errorCount, cpuPercentile, memoryPercentile, poolFallbackCount, waitDurationCount, resourceConsumptionAboveThreshold)
	return &PrometheusMetrics{
		TaskCompletedCount:                taskCompletedCount,
		TaskFailedCount:                   taskFailedCount,
		TaskRunningCount:                  taskRunningCount,
		TaskTimeoutCount:                  taskTimeoutCount,
		TaskExecutionTime:                 taskExecutionTime,
		HeartbeatFailureCount:             heartbeatFailureCount,
		ErrorCount:                        errorCount,
		MemoryPercentile:                  memoryPercentile,
		CPUPercentile:                     cpuPercentile,
		PoolFallbackCount:                 poolFallbackCount,
		WaitDurationCount:                 waitDurationCount,
		ResourceConsumptionAboveThreshold: resourceConsumptionAboveThreshold,
	}
}

func NewPrometheusMetrics() metrics.Metrics {
	return registerMetrics()
}
