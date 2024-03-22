// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package poller

import (
	"context"
	"sync"
	"time"

	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/router"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	taskEventsTimeout = 30 * time.Second
)

type FilterFn func(*client.RunnerEvent) bool

type EventsServer struct {
	Client client.Client
	router router.Router
	Filter FilterFn
	// The Harness manager allows two task acquire calls with the same delegate ID to go through (by design).
	// We need to make sure two different threads do not acquire the same task.
	// This map makes sure Acquire() is called only once per task ID. The mapping is removed once the status
	// for the task has been sent.
	m sync.Map
}

func New(c client.Client, router router.Router) *EventsServer {
	return &EventsServer{
		Client: c,
		router: router,
		m:      sync.Map{},
	}
}

func (p *EventsServer) SetFilter(filter FilterFn) {
	p.Filter = filter
}

// Poll continually asks the task server for tasks to execute.
func (p *EventsServer) PollRunnerEvents(ctx context.Context, n int, id string, interval time.Duration) error {
	//	var wg sync.WaitGroup
	events := make(chan *client.RunnerEvent, n)
	// Task event poller
	go func() {
		pollTimer := time.NewTimer(interval)
		for {
			pollTimer.Reset(interval)
			select {
			case <-ctx.Done():
				logrus.Error("context canceled")
				return
			case <-pollTimer.C:
				taskEventsCtx, cancelFn := context.WithTimeout(ctx, taskEventsTimeout)
				tasks, err := p.Client.GetRunnerEvents(taskEventsCtx, id)
				if err != nil {
					logrus.WithError(err).Errorf("could not query for task events")
				}
				cancelFn()

				for _, e := range tasks.RunnerEvents {
					events <- e
				}
			}
		}
	}()
	// Task event processor. Start n threads to process events from the channel
	for i := 0; i < n; i++ {
		go func(i int) {
			for {
				select {
				case <-ctx.Done():
					return
				case task := <-events:
					err := p.process(ctx, id, *task)
					if err != nil {
						logrus.WithError(err).WithField("task_id", task.TaskID).Errorf("[Thread %d]: delegate [%s] could not process runner request", i, id)
					}
				}
			}
		}(i)
	}
	logrus.Infof("initialized %d threads successfully and starting polling for tasks", n)
	return nil
}

// execute tries to acquire the task and executes the handler for it
func (p *EventsServer) process(ctx context.Context, delegateID string, rv client.RunnerEvent) error {
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
		taskResponse := &client.TaskResponse{ID: request.ID, Type: "RUNNER_TASK"}
		if resp.Error() != nil {
			taskResponse.Code = "FAILED"
		} else {
			taskResponse.Code = "OK"
			taskResponse.Data = resp.Body()
		}
		if err := p.Client.SendStatus(ctx, delegateID, request.ID, taskResponse); err != nil {
			return err
		}
	}
	return nil
}
