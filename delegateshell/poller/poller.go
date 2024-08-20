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

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/runner/delegateshell/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	taskEventsTimeout = 30 * time.Second
)

type FilterFn func(*client.RunnerEvent) bool

type Poller struct {
	UseV2Status bool
	Client      client.Client
	router      *task.Router
	Filter      FilterFn
	// The Harness manager allows two task acquire calls with the same delegate ID to go through (by design).
	// We need to make sure two different threads do not acquire the same task.
	// This map makes sure Acquire() is called only once per task ID. The mapping is removed once the status
	// for the task has been sent.
	m sync.Map
}

func New(c client.Client, router *task.Router, useV2 bool) *Poller {
	return &Poller{
		Client:      c,
		router:      router,
		m:           sync.Map{},
		UseV2Status: useV2,
	}
}

func (p *Poller) SetFilter(filter FilterFn) {
	p.Filter = filter
}

// PollRunnerEvents Poll continually asks the task server for tasks to execute.
func (p *Poller) PollRunnerEvents(ctx context.Context, n int, id string, interval time.Duration) error {

	events := make(chan *client.RunnerEvent, n)
	var wg sync.WaitGroup

	// Task event poller
	go func() {
		defer close(events) // Ensure the events channel is closed when polling stops
		pollTimer := time.NewTimer(interval)
		defer pollTimer.Stop()

		for {
			pollTimer.Reset(interval)
			select {
			case <-ctx.Done():
				logrus.Error("context canceled, stopping task polling")
				return
			case <-pollTimer.C:
				taskEventsCtx, cancelFn := context.WithTimeout(ctx, taskEventsTimeout)
				tasks, err := p.Client.GetRunnerEvents(taskEventsCtx, id)
				if err != nil {
					logrus.WithError(err).Errorf("could not query for task events")
				}
				cancelFn()

				for _, e := range tasks.RunnerEvents {
					select {
					case events <- e:
						// Event successfully sent to the channel
					case <-ctx.Done():
						logrus.Info("context canceled during event processing")
						// Context canceled, but let the loop exit naturally so that all acquired events are processed
						return
						// TODO check if this causes premature return without processing all the events
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
				err := p.process(ctx, id, *acquiredTask)
				if err != nil {
					logrus.WithError(err).WithField("task_id", acquiredTask.TaskID).Errorf("[Thread %d]: runner [%s] could not process request", i, id)
				}
			}
		}(i)
	}
	logrus.Infof("initialized %d threads successfully and starting polling for tasks", n)
	wg.Wait()
	return nil
}

// execute tries to acquire the task and executes the handler for it
func (p *Poller) process(ctx context.Context, delegateID string, rv client.RunnerEvent) error {
	taskID := rv.TaskID
	if _, loaded := p.m.LoadOrStore(taskID, true); loaded {
		return nil
	}
	defer p.m.Delete(taskID)
	payloads, err := p.Client.GetExecutionPayload(ctx, delegateID, taskID)
	if err != nil {
		return errors.Wrap(err, "failed to get payload")
	}
	// Since task id is unique, it's just one request
	for _, request := range payloads.Requests {
		resp := p.router.Handle(ctx, request)
		if resp == nil {
			continue
		}
		taskResponse := &client.TaskResponse{ID: rv.TaskID, Type: "CI_EXECUTE_STEP"}
		if resp.Error() != nil {
			taskResponse.Code = "FAILED"
			logrus.WithError(resp.Error()).Error("Process task failed")
			// TODO: a bug here. If the Data is nil, exception happen in cg manager.
			// This will be taken care after integrating with new response workflow
			if respBytes, err := json.Marshal(&api.VMTaskExecutionResponse{ErrorMessage: resp.Error().Error()}); err != nil {
				taskResponse.Data = respBytes
			} else {
				return err
			}
		} else {
			taskResponse.Code = "OK"
			taskResponse.Data = resp.Body()
		}
		if p.UseV2Status {
			taskResponseV2 := &client.TaskResponseV2{
				ID:   taskResponse.ID,
				Data: taskResponse.Data,
				Type: request.Task.Type,
				Code: client.StatusCode(taskResponse.Code),
			}
			if err := p.Client.SendStatusV2(ctx, delegateID, rv.TaskID, taskResponseV2); err != nil {
				return err
			}
		} else {
			if err := p.Client.SendStatus(ctx, delegateID, rv.TaskID, taskResponse); err != nil {
				return err
			}
		}
	}
	return nil
}
