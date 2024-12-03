package metrics

import (
	"errors"

	"github.com/drone-runners/drone-runner-aws/metric"
	"github.com/harness/runner/metrics"
	"github.com/harness/runner/metrics/providers/prometheus"
)

// The VM runner today uses a metrics object to do metric reporting.
// Once the runner is adopted fully, we can standardize on using an interface.
// Instead of refactoring the VM runner everywhere to use an interface,
// for now we will just create a compatible metrics object.
func ProvideVMMetrics(m metrics.Metrics) (*metric.Metrics, error) {
	if pm, ok := m.(*prometheus.PrometheusMetrics); ok {
		return &metric.Metrics{
			BuildCount:             pm.PipelineExecutionTotalCount,
			FailedCount:            pm.PipelineExecutionErrorsTotalCount,
			ErrorCount:             pm.PipelineSystemErrorsTotalCount,
			RunningCount:           pm.PipelineExecutionsRunning,
			RunningPerAccountCount: pm.PipelineExecutionsRunningPerAccount,
			PoolFallbackCount:      pm.PipelinePoolFallbackCount,
			WaitDurationCount:      pm.PipelineWaitDurationTime,
			CPUPercentile:          pm.PipelineMaxCPUPercentile,
			MemoryPercentile:       pm.PipelineMaxMemoryPercentile,
		}, nil
	}
	return &metric.Metrics{}, errors.New("unsupported metrics provider for vm based builds")
}
