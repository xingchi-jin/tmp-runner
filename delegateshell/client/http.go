// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/utils"
	"github.com/sirupsen/logrus"
)

const (
	registerEndpoint         = "/api/agent/delegates/register?accountId=%s"
	heartbeatEndpoint        = "/api/agent/delegates/heartbeat-with-polling?accountId=%s"
	taskStatusEndpoint       = "/api/agent/v2/tasks/%s/delegates/%s?accountId=%s"
	runnerEventsPollEndpoint = "/api/executions/%s/runner-events?accountId=%s"
	executionPayloadEndpoint = "/api/executions/%s/request?delegateId=%s&accountId=%s&delegateInstanceId=%s"
	taskStatusEndpointV2     = "/api/executions/%s/response?delegateId=%s&accountId=%s"
)

var (
	registerTimeout      = 30 * time.Second
	taskEventsTimeout    = 60 * time.Second
	sendStatusRetryTimes = 5
)

type ManagerClient struct {
	utils.HTTPClient
	AccountID  string
	Token      string
	TokenCache *delegate.TokenCache
}

func NewManagerClient(endpoint, id, secret string, skipverify bool, additionalCertsDir string) *ManagerClient {
	return &ManagerClient{
		HTTPClient: *utils.New(endpoint, skipverify, additionalCertsDir),
		AccountID:  id,
		TokenCache: delegate.NewTokenCache(id, secret),
	}
}

// Register registers the runner with the manager
func (p *ManagerClient) Register(ctx context.Context, r *RegisterRequest) (*RegisterResponse, error) {
	req := r
	resp := &RegisterResponse{}
	path := fmt.Sprintf(registerEndpoint, p.AccountID)
	_, err := p.retry(ctx, path, "POST", req, resp, createBackoff(ctx, registerTimeout), true) //nolint: bodyclose
	return resp, err
}

// Heartbeat sends a periodic heartbeat to the server
func (p *ManagerClient) Heartbeat(ctx context.Context, r *RegisterRequest) error {
	req := r
	path := fmt.Sprintf(heartbeatEndpoint, p.AccountID)
	_, err := p.doJson(ctx, path, "POST", req, nil)
	return err
}

// GetRunnerEvents gets a list of events which can be executed on this runner
func (p *ManagerClient) GetRunnerEvents(ctx context.Context, id string) (*RunnerEventsResponse, error) {
	path := fmt.Sprintf(runnerEventsPollEndpoint, id, p.AccountID)
	events := &RunnerEventsResponse{}
	_, err := p.doJson(ctx, path, "GET", nil, events)
	return events, err
}

// Acquire tries to acquire a specific task
func (p *ManagerClient) GetExecutionPayload(ctx context.Context, delegateID, taskID string) (*RunnerAcquiredTasks, error) {
	path := fmt.Sprintf(executionPayloadEndpoint, taskID, delegateID, p.AccountID, delegateID)
	payload := &RunnerAcquiredTasks{}
	_, err := p.doJson(ctx, path, "GET", nil, payload)
	if err != nil {
		logrus.WithError(err).Error("Error making http call")
	}
	return payload, err
}

// SendStatus updates the status of a task
func (p *ManagerClient) SendStatus(ctx context.Context, delegateID, taskID string, r *TaskResponse) error {
	path := fmt.Sprintf(taskStatusEndpoint, taskID, delegateID, p.AccountID)
	req := r
	retryNumber := 0
	var err error
	for retryNumber < sendStatusRetryTimes {
		_, err = p.retry(ctx, path, "POST", req, nil, createBackoff(ctx, taskEventsTimeout), true) //nolint: bodyclose
		if err == nil {
			return nil
		}
		retryNumber++
	}
	return err
}

// SendStatusV2 updates the status of a task using the v2 endpoint which submits task
// responses via events framework.
func (p *ManagerClient) SendStatusV2(ctx context.Context, delegateID, taskID string, r *TaskResponseV2) error {
	path := fmt.Sprintf(taskStatusEndpointV2, taskID, delegateID, p.AccountID)
	req := r
	_, err := p.doJson(ctx, path, "POST", req, nil)
	return err
}

func (p *ManagerClient) retry(ctx context.Context, path, method string, in, out interface{}, b backoff.BackOffContext, ignoreStatusCode bool) (*http.Response, error) { //nolint: unparam
	for {
		res, err := p.doJson(ctx, path, method, in, out)
		// do not retry on Canceled or DeadlineExceeded
		if ctxErr := ctx.Err(); ctxErr != nil {
			p.Logger.Errorf("http: context canceled")
			return res, ctxErr
		}

		duration := b.NextBackOff()

		if res != nil {
			// Check the response code. We retry on 500-range
			// responses to allow the server time to recover, as
			// 500's are typically not permanent errors and may
			// relate to outages on the server side.
			if (ignoreStatusCode && err != nil) || res.StatusCode > 501 {
				p.Logger.Errorf("url: %s server error: re-connect and re-try: %s", path, err)
				if duration == backoff.Stop {
					p.Logger.Errorf("max retry limit reached, task status won't be updated")
					return nil, err
				}
				time.Sleep(duration)
				continue
			}
		} else if err != nil {
			p.Logger.Errorf("http: request error: %s", err)
			if duration == backoff.Stop {
				p.Logger.Errorf("max retry limit reached, task status won't be updated")
				return nil, err
			}
			time.Sleep(duration)
			continue
		}
		return res, err
	}
}

func (p *ManagerClient) doJson(ctx context.Context, path, method string, in, out interface{}) (*http.Response, error) {
	var buf = &bytes.Buffer{}
	// marshal the input payload into json format and copy
	// to an io.ReadCloser.
	if in != nil {
		if err := json.NewEncoder(buf).Encode(in); err != nil {
			p.Logger.Errorf("could not encode input payload: %s", err)
		}
	}
	// the request should include the secret shared between
	// the agent and server for authorization.
	var err error
	token := ""
	if p.Token != "" {
		token = p.Token
	} else {
		token, err = p.TokenCache.Get()
		if err != nil {
			p.Logger.Errorf("could not generate account token: %s", err)
			return nil, err
		}
	}
	headers := make(map[string]string)
	headers["Authorization"] = "Delegate " + token
	headers["Content-Type"] = "application/json"
	res, body, err := p.Do(ctx, path, method, headers, buf)
	if err != nil {
		return res, err
	}
	if nil == out {
		return res, nil
	}
	if jsonErr := json.Unmarshal(body, out); jsonErr != nil {
		return res, jsonErr
	}

	return res, nil
}

func createBackoff(ctx context.Context, maxElapsedTime time.Duration) backoff.BackOffContext {
	exp := backoff.NewExponentialBackOff()
	exp.MaxElapsedTime = maxElapsedTime
	return backoff.WithContext(exp, ctx)
}
