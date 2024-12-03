// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prometheus

import (
	"github.com/drone-runners/drone-runner-aws/metric"
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
	TaskRejectedCount                 *prometheus.CounterVec
	ResourceConsumptionAboveThreshold *prometheus.GaugeVec

	// CI based metrics
	PipelineExecutionTotalCount         *prometheus.CounterVec
	PipelineSystemErrorsTotalCount      *prometheus.CounterVec
	PipelineExecutionErrorsTotalCount   *prometheus.CounterVec
	PipelineExecutionsRunning           *prometheus.GaugeVec
	PipelineExecutionsRunningPerAccount *prometheus.GaugeVec
	PipelinePoolFallbackCount           *prometheus.CounterVec
	PipelineWaitDurationTime            *prometheus.HistogramVec
	PipelineMaxCPUPercentile            *prometheus.HistogramVec
	PipelineMaxMemoryPercentile         *prometheus.HistogramVec
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

// --------------------------------------------------------------------------------------------------------------------------------------
// CI based metrics
// Metric names need to be same until dlite is completely replaced by the runner to avoid
// partial reporting under different names.
// We use the VM runner metrics to initialize directly.
// --------------------------------------------------------------------------------------------------------------------------------------

// PipelineExecutionTotalCount provides metrics for total number of pipeline executions (failed + successful)
func PipelineExecutionTotalCount() *prometheus.CounterVec {
	return metric.BuildCount()
}

// PipelineSystemErrorsTotalCount provides metrics for total number of system errors
func PipelineSystemErrorsTotalCount() *prometheus.CounterVec {
	return metric.ErrorCount()
}

// PipelineExecutionErrorsTotalCount provides metrics for total number of failed pipeline executions
func PipelineExecutionErrorsTotalCount() *prometheus.CounterVec {
	return metric.FailedBuildCount()
}

// PipelineExecutionsRunning provides metrics for number of executing pipelines
func PipelineExecutionsRunning() *prometheus.GaugeVec {
	return metric.RunningCount()
}

// PipelineExecutionsRunningPerAccount provides metrics for number of executing pipelines per account
func PipelineExecutionsRunningPerAccount() *prometheus.GaugeVec {
	return metric.RunningPerAccountCount()
}

// PipelinePoolFallbackCount provides metrics for number of fallbacks while finding a valid pool
func PipelinePoolFallbackCount() *prometheus.CounterVec {
	return metric.PoolFallbackCount()
}

// PipelineMaxCPUPercentile provides information about the max CPU usage in the pipeline run
func PipelineMaxCPUPercentile() *prometheus.HistogramVec {
	return metric.CPUPercentile()
}

// PipelineMaxMemoryPercentile provides information about the max RAM usage in the pipeline run
func PipelineMaxMemoryPercentile() *prometheus.HistogramVec {
	return metric.MemoryPercentile()
}

// PipelineWaitDuration provides metrics for amount of time needed to wait to setup a machine
func PipelineWaitDurationTime() *prometheus.HistogramVec {
	return metric.WaitDurationCount()
}

// registerMetrics sets up the metrics client and returns the Metrics object.
func registerMetrics() metrics.Metrics {
	taskCompletedCount := TaskCompletedCount()
	taskFailedCount := TaskFailedCount()
	taskRunningCount := TaskRunningCount()
	taskTimeoutCount := TaskTimeoutCount()
	taskExecutionTime := TaskExecutionTime()
	heartbeatFailureCount := HeartbeatFailureCount()
	resourceConsumptionAboveThreshold := ResourceConsumptionAboveThreshold()
	errorCount := ErrorCount()

	// CI based metrics
	pipelineExecutionCount := PipelineExecutionTotalCount()
	pipelineSystemErrorsTotalCount := PipelineSystemErrorsTotalCount()
	pipelineExecutionErrorsCount := PipelineExecutionErrorsTotalCount()
	pipelineExecutionsRunning := PipelineExecutionsRunning()
	pipelineExecutionsRunningPerAccount := PipelineExecutionsRunningPerAccount()
	pipelinePoolFallbackCount := PipelinePoolFallbackCount()
	pipelineWaitDurationTime := PipelineWaitDurationTime()
	pipelineMaxCPUPercentile := PipelineMaxCPUPercentile()
	pipelineMaxMemoryPercentile := PipelineMaxMemoryPercentile()

	prometheus.MustRegister(taskCompletedCount, taskFailedCount, taskRunningCount, taskTimeoutCount, taskExecutionTime, heartbeatFailureCount, resourceConsumptionAboveThreshold, errorCount,
		pipelineExecutionCount, pipelineExecutionErrorsCount, pipelineExecutionsRunning, pipelineExecutionsRunningPerAccount, pipelinePoolFallbackCount, pipelineWaitDurationTime,
		pipelineMaxCPUPercentile, pipelineMaxMemoryPercentile, pipelineSystemErrorsTotalCount)
	return &PrometheusMetrics{
		TaskCompletedCount:                  taskCompletedCount,
		TaskFailedCount:                     taskFailedCount,
		TaskRunningCount:                    taskRunningCount,
		TaskTimeoutCount:                    taskTimeoutCount,
		TaskExecutionTime:                   taskExecutionTime,
		HeartbeatFailureCount:               heartbeatFailureCount,
		ErrorCount:                          errorCount,
		ResourceConsumptionAboveThreshold:   resourceConsumptionAboveThreshold,
		PipelineSystemErrorsTotalCount:      pipelineSystemErrorsTotalCount,
		PipelineExecutionTotalCount:         pipelineExecutionCount,
		PipelineExecutionErrorsTotalCount:   pipelineExecutionErrorsCount,
		PipelineExecutionsRunning:           pipelineExecutionsRunning,
		PipelineExecutionsRunningPerAccount: pipelineExecutionsRunningPerAccount,
		PipelinePoolFallbackCount:           pipelinePoolFallbackCount,
		PipelineWaitDurationTime:            pipelineWaitDurationTime,
		PipelineMaxCPUPercentile:            pipelineMaxCPUPercentile,
		PipelineMaxMemoryPercentile:         pipelineMaxMemoryPercentile,
	}
}

func NewPrometheusMetrics() metrics.Metrics {
	return registerMetrics()
}
