// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metrics

type Metrics interface {
	// Task-based Metrics
	IncrementTaskCompletedCount(accountID, taskType, runnerName string)
	IncrementTaskFailedCount(accountID, taskType, runnerName string)
	IncrementTaskRunningCount(accountID, taskType, runnerName string)
	DecrementTaskRunningCount(accountID, taskType, runnerName string)
	IncrementTaskTimeoutCount(accountID, taskType, runnerName string) // Not implemented
	SetTaskExecutionTime(accountID, taskType, runnerName, taskID string, executionTime float64)
	// Runner Metrics
	IncrementHeartbeatFailureCount(accountID, runnerName string)
	IncrementErrorCount(accountID, runnerName string)                      // Not implemented
	SetResourceConsumptionIsAboveThreshold(accountID, runnerName string)   // Not implemented
	UnsetResourceConsumptionIsAboveThreshold(accountID, runnerName string) // Not implemented
	// VM Task based Metrics
	SetCPUPercentile(accountID, runnerName string, cpuPercentage float64)                 // Not implemented
	SetMemoryPercentile(accountID, runnerName string, cpuPercentage float64)              // Not implemented
	IncrementPoolFallbackCount(accountID, runnerName, success string)                     // Not implemented
	ObserveWaitDurationCount(accountID, runnerName, isFallback string, setupTime float64) // Not implemented
}

const MetricNamePrefix = "io_harness"
