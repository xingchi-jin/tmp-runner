package metrics

import (
	"context"
	"errors"

	"github.com/drone-runners/drone-runner-aws/metric"
	"github.com/drone-runners/drone-runner-aws/store"
	"github.com/drone-runners/drone-runner-aws/types"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/metrics"
	"github.com/harness/runner/metrics/providers/prometheus"
)

// The VM runner today uses a metrics object to do metric reporting.
// Once the runner is adopted fully, we can standardize on using an interface.
// Instead of refactoring the VM runner everywhere to use an interface,
// for now we will just create a compatible metrics object.
func ProvideVMMetrics(
	ctx context.Context,
	m metrics.Metrics,
	instanceStore store.InstanceStore,
	config *delegate.Config,
) (*metric.Metrics, error) {
	if config.VM.Pool.File == "" {
		return nil, nil
	}
	if pm, ok := m.(*prometheus.PrometheusMetrics); ok {
		m := &metric.Metrics{
			BuildCount:             pm.PipelineExecutionTotalCount,
			FailedCount:            pm.PipelineExecutionErrorsTotalCount,
			ErrorCount:             pm.PipelineSystemErrorsTotalCount,
			RunningCount:           pm.PipelineExecutionsRunning,
			RunningPerAccountCount: pm.PipelineExecutionsRunningPerAccount,
			PoolFallbackCount:      pm.PipelinePoolFallbackCount,
			WaitDurationCount:      pm.PipelineWaitDurationTime,
			CPUPercentile:          pm.PipelineMaxCPUPercentile,
			MemoryPercentile:       pm.PipelineMaxMemoryPercentile,
		}
		m.AddMetricStore(&metric.Store{
			Store: instanceStore,
			Query: &types.QueryParams{
				RunnerName: config.GetName(),
			},
			Distributed: true,
		})
		m.UpdateRunningCount(ctx)
		return m, nil
	}
	return nil, errors.New("unsupported metrics provider for vm based builds")
}
