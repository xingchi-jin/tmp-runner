// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prometheus

func (p *PrometheusMetrics) IncrementTaskCompletedCount(accountID, taskType, runnerName string) {
	p.TaskCompletedCount.WithLabelValues(accountID, taskType, runnerName).Inc()
}

func (p *PrometheusMetrics) IncrementTaskFailedCount(accountID, taskType, runnerName string) {
	p.TaskFailedCount.WithLabelValues(accountID, taskType, runnerName).Inc()
}

func (p *PrometheusMetrics) IncrementTaskRunningCount(accountID, taskType, runnerName string) {
	p.TaskRunningCount.WithLabelValues(accountID, taskType, runnerName).Inc()
}

func (p *PrometheusMetrics) DecrementTaskRunningCount(accountID, taskType, runnerName string) {
	p.TaskRunningCount.WithLabelValues(accountID, taskType, runnerName).Dec()
}

func (p *PrometheusMetrics) IncrementTaskTimeoutCount(accountID, taskType, runnerName string) {
	p.TaskTimeoutCount.WithLabelValues(accountID, taskType, runnerName).Inc()
}

func (p *PrometheusMetrics) SetTaskExecutionTime(accountID, taskType, runnerName, taskID string, executionTime float64) {
	p.TaskExecutionTime.WithLabelValues(accountID, taskType, taskID, runnerName).Set(executionTime)
}

func (p *PrometheusMetrics) IncrementHeartbeatFailureCount(accountID, runnerName string) {
	p.HeartbeatFailureCount.WithLabelValues(accountID, runnerName).Inc()
}

func (p *PrometheusMetrics) IncrementErrorCount(accountID, runnerName string) {
	p.TaskTimeoutCount.WithLabelValues(accountID, runnerName).Inc()
}

func (p *PrometheusMetrics) SetResourceConsumptionIsAboveThreshold(accountID, runnerName string) {
	p.ResourceConsumptionAboveThreshold.WithLabelValues(accountID, runnerName).Set(1)
}

func (p *PrometheusMetrics) UnsetResourceConsumptionIsAboveThreshold(accountID, runnerName string) {
	p.ResourceConsumptionAboveThreshold.WithLabelValues(accountID, runnerName).Set(0)
}
