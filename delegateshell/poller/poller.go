// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package poller

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/harness/runner/logger"
	"github.com/harness/runner/metrics"
	metricsutils "github.com/harness/runner/metrics/utils"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/runner/delegateshell/client"
	"github.com/pkg/errors"
)

var (
	taskEventsTimeout = 30 * time.Second
)

type FilterFn func(*client.RunnerEvent) bool

type Poller struct {
	UseV2Status   bool
	RemoteLogging bool
	Client        client.Client
	router        *task.Router
	Metrics       metrics.Metrics
	Filter        FilterFn
	stopChannel   chan struct{}
	doneChannel   chan struct{}
	// The Harness manager allows two task acquire calls with the same delegate ID to go through (by design).
	// We need to make sure two different threads do not acquire the same task.
	// This map makes sure Acquire() is called only once per task ID. The mapping is removed once the status
	// for the task has been sent.
	m sync.Map
}

func New(c client.Client, router *task.Router, metrics metrics.Metrics, remoteLogging bool) *Poller {
	p := &Poller{
		Client:        c,
		router:        router,
		Metrics:       metrics,
		m:             sync.Map{},
		RemoteLogging: remoteLogging,
	}
	p.stopChannel = make(chan struct{})
	p.doneChannel = make(chan struct{})
	return p
}

func (p *Poller) SetFilter(filter FilterFn) {
	p.Filter = filter
}

// PollRunnerEvents continually asks the task server for tasks to execute.
func (p *Poller) PollRunnerEvents(ctx context.Context, n int, id, name string, interval time.Duration) error {

	events := make(chan *client.RunnerEvent, n)
	var wg sync.WaitGroup

	// Task event poller
	go func() {
		defer close(events)
		pollTimer := time.NewTimer(interval)
		defer pollTimer.Stop()

		for {
			pollTimer.Reset(interval)
			select {
			case <-ctx.Done():
				logger.Errorln(ctx, "context canceled during task polling, this should not happen")
				return
			case <-p.stopChannel:
				logger.Infoln(ctx, "Request received to stop the poller")
				// Note: The goal here is to stop the poller from acquiring new tasks,
				// but we want to allow any ongoing tasks that were already in progress to complete.
				if !pollTimer.Stop() {
					// We attempt to stop the timer. If `pollTimer.Stop()` returns `false`,
					// it means the timer was either already triggered or is currently firing.
					// In this case, there might be an event pending on the channel `pollTimer.C`.
					// To ensure that no timer events are left unprocessed before stopping the poller,
					// we need to drain the channel by waiting to receive the event from `pollTimer.C`

					logger.Infoln(ctx, "Waiting for any ongoing events to complete")
					<-pollTimer.C
				}
				logger.Infoln(ctx, "Task polling has been stopped")
				return
			case <-pollTimer.C:
				taskEventsCtx, cancelFn := context.WithTimeout(ctx, taskEventsTimeout)
				tasks, err := p.Client.GetRunnerEvents(taskEventsCtx, id)
				if err != nil {
					logger.WithError(ctx, err).Errorf("could not query for task events")
				}
				cancelFn()

				for _, e := range tasks.RunnerEvents {
					select {
					case events <- e:
						// Event successfully sent to the channel
					case <-ctx.Done():
						logger.Errorln(ctx, "context canceled during event processing, this should not happen")
						return
					}
				}
			}
		}
	}()
	// Task event processor. Start n threads to process events from the channel
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for acquiredTask := range events { // Read from events channel until it's closed
				ctx = logger.AddLogLabelsToContext(ctx, map[string]string{"task_id": acquiredTask.TaskID})
				err := p.process(ctx, id, name, *acquiredTask)
				if err != nil {
					logger.WithError(ctx, err).Errorf("[Thread %d]: runner [%s] could not process request", i, id)
				}
			}
		}(i)
	}
	logger.Infof(ctx, "Initialized %d threads successfully and starting polling for tasks", n)
	wg.Wait()
	// After all tasks are processed, notify completion
	close(p.doneChannel)
	return nil
}

// execute tries to acquire the task and executes the handler for it
func (p *Poller) process(ctx context.Context, delegateID, delegateName string, rv client.RunnerEvent) error {
	taskID := rv.TaskID
	if _, loaded := p.m.LoadOrStore(taskID, true); loaded {
		return nil
	}
	defer p.m.Delete(taskID)

	payloads, err := p.Client.GetExecutionPayload(ctx, delegateID, delegateName, taskID)
	if err != nil {
		return errors.Wrap(err, "failed to get payload")
	}
	// Since task id is unique, it's just one request
	for _, request := range payloads.Requests {
		p.Metrics.IncrementTaskRunningCount(rv.AccountID, rv.TaskType, delegateName)
		defer p.Metrics.DecrementTaskRunningCount(rv.AccountID, rv.TaskType, delegateName)
		start_time := time.Now()

		// TODO set the task id in runner request translator
		// task id is required by the lite engine to send the response to the manager for hosted builds
		request.Task.ID = rv.TaskID
		resp := p.router.Handle(ctx, request)
		p.Metrics.SetTaskExecutionTime(rv.AccountID, rv.TaskType, rv.TaskID, delegateName, metricsutils.CalculateDuration(start_time))
		if resp == nil {
			continue
		}
		taskResponse := &client.TaskResponse{ID: rv.TaskID, Type: request.Task.Type}
		p.Metrics.IncrementTaskCompletedCount(rv.AccountID, rv.TaskType, delegateName)

		if resp.Error() != nil {
			taskResponse.Code = client.StatusCodeFailed
			logger.WithError(ctx, resp.Error()).Error("Process task failed")
			taskResponse.Error = resp.Error().Error()
			p.Metrics.IncrementTaskFailedCount(rv.AccountID, rv.TaskType, delegateName)
			// TODO: a bug here. If the Data is nil, exception happen in cg manager.
			// This will be taken care after integrating with new response workflow
			if respBytes, err := json.Marshal(&api.VMTaskExecutionResponse{ErrorMessage: resp.Error().Error()}); err == nil {
				taskResponse.Data = respBytes
			} else {
				return err
			}
		} else {
			taskResponse.Code = client.StatusCodeSuccess
			taskResponse.Data = resp.Body()
		}
		if err := p.Client.SendStatus(ctx, delegateID, rv.TaskID, taskResponse); err != nil {
			return err
		}
	}
	return nil
}

func (p *Poller) Shutdown(ctx context.Context) {
	p.stopPollingForTasks()
	logger.Infoln(ctx, "Notified poller to stop acquiring new tasks, waiting for in progress tasks completion")
	p.waitForTasks()
	logger.Infoln(ctx, "All tasks are completed, stopping task processor...")
}

func (p *Poller) stopPollingForTasks() {
	close(p.stopChannel) // Notify poller to stop acquiring new tasks
}

func (p *Poller) waitForTasks() {
	<-p.doneChannel // Wait for all tasks to be processed
}
