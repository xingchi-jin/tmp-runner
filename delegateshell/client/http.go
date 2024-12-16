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

	"github.com/harness/runner/logger"

	"github.com/cenkalti/backoff/v4"
	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/utils"
)

const (
	registerEndpoint                = "/api/agent/delegates/register?accountId=%s"
	unregisterEndpoint              = "/api/agent/delegates/unregister?accountId=%s"
	heartbeatEndpoint               = "/api/agent/delegates/heartbeat-with-polling?accountId=%s"
	taskStatusEndpoint              = "/api/agent/v2/tasks/%s/delegates/%s?accountId=%s"
	runnerEventsPollEndpoint        = "/api/executions/%s/runner-events?accountId=%s"
	executionPayloadEndpoint        = "/api/executions/%s/request?delegateId=%s&accountId=%s&delegateInstanceId=%s&delegateName=%s"
	taskStatusEndpointV2            = "/api/executions/%s/task-response?runnerId=%s&accountId=%s"
	daemonSetReconcileEndpoint      = "/api/daemons/%s/reconcile?accountId=%s"
	acquireDaemonTasksEndpoint      = "/api/daemons/%s/tasks?accountId=%s"
	stackDriverLoggingTokenEndpoint = "/api/agent/infra-download/delegate-auth/delegate/logging-token?accountId=%s"
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

func NewManagerClient(endpoint, accountID, secret string, skipverify bool, additionalCertsDir string) *ManagerClient {
	return &ManagerClient{
		HTTPClient: *utils.New(endpoint, skipverify, additionalCertsDir),
		AccountID:  accountID,
		TokenCache: delegate.NewTokenCache(accountID, secret),
	}
}

// ReconcileDaemonSets calls the daemon set reconciliation endpoint in manager
func (p *ManagerClient) ReconcileDaemonSets(ctx context.Context, runnerId string, r *DaemonSetReconcileRequest) (*DaemonSetReconcileResponse, error) {
	req := r
	resp := &DaemonSetReconcileResponse{}
	path := fmt.Sprintf(daemonSetReconcileEndpoint, runnerId, p.AccountID)
	_, err := p.doJson(ctx, path, "POST", req, resp)
	return resp, err
}

// AcquireDaemonTasks fetches daemon task data from manager
func (p *ManagerClient) AcquireDaemonTasks(ctx context.Context, runnerId string, r *DaemonTaskAcquireRequest) (*RunnerAcquiredTasks, error) {
	req := r
	resp := &RunnerAcquiredTasks{}
	path := fmt.Sprintf(acquireDaemonTasksEndpoint, runnerId, p.AccountID)
	_, err := p.retry(ctx, path, "POST", req, resp, createBackoff(ctx, registerTimeout), false) //nolint: bodyclose
	return resp, err
}

// Register registers the runner with the manager
func (p *ManagerClient) Register(ctx context.Context, r *RegisterRequest) (*RegisterResponse, error) {
	req := r
	resp := &RegisterResponse{}
	path := fmt.Sprintf(registerEndpoint, p.AccountID)
	_, err := p.retry(ctx, path, "POST", req, resp, createBackoff(ctx, registerTimeout), true) //nolint: bodyclose
	return resp, err
}

// Unregister unregisters the runner with the manager
func (p *ManagerClient) Unregister(ctx context.Context, r *UnregisterRequest) error {
	req := r
	path := fmt.Sprintf(unregisterEndpoint, p.AccountID)
	_, err := p.retry(ctx, path, "POST", req, nil, createBackoff(ctx, registerTimeout), true)
	return err
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
func (p *ManagerClient) GetExecutionPayload(ctx context.Context, delegateID, delegateName, taskID string) (*RunnerAcquiredTasks, error) {
	path := fmt.Sprintf(executionPayloadEndpoint, taskID, delegateID, p.AccountID, delegateID, delegateName)
	payload := &RunnerAcquiredTasks{}
	_, err := p.doJson(ctx, path, "GET", nil, payload)
	if err != nil {
		logger.WithError(ctx, err).Error("Error making http call")
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
			logger.Errorf(ctx, "http: context canceled")
			return res, ctxErr
		}

		duration := b.NextBackOff()

		if res != nil {
			// Check the response code. We retry on 500-range
			// responses to allow the server time to recover, as
			// 500's are typically not permanent errors and may
			// relate to outages on the server side.
			if (ignoreStatusCode && err != nil) || res.StatusCode > 501 {
				logger.Errorf(ctx, "url: %s server error: re-connect and re-try: %s", path, err)
				if duration == backoff.Stop {
					logger.Errorf(ctx, "max retry limit reached, task status won't be updated")
					return nil, err
				}
				time.Sleep(duration)
				continue
			}
		} else if err != nil {
			logger.Errorf(ctx, "http: request error: %s", err)
			if duration == backoff.Stop {
				logger.Errorf(ctx, "max retry limit reached, task status won't be updated")
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
			logger.Errorf(ctx, "could not encode input payload: %s", err)
		}
	}
	// the request should include the secret shared between
	// the agent and server for authorization.
	var err error
	token := ""
	if p.Token != "" {
		token = p.Token
	} else {
		token, err = p.TokenCache.Get(ctx)
		if err != nil {
			logger.Errorf(ctx, "could not generate account token: %s", err)
			return nil, err
		}
	}
	headers := make(map[string]string)
	headers["Authorization"] = "Delegate " + token
	headers["Content-Type"] = "application/json"
	headers["delegateTokenHash"] = p.TokenCache.GetTokenHash()
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

func (p *ManagerClient) GetLoggingToken(ctx context.Context) (*AccessTokenBean, error) {
	path := fmt.Sprintf(stackDriverLoggingTokenEndpoint, p.AccountID)
	credentials := &AccessTokenBeanResource{}
	_, err := p.doJson(ctx, path, "GET", nil, credentials)
	if err != nil {
		logger.WithError(ctx, err).Error("Error getting stack driver logging token")
	}
	return credentials.AccessTokenBean, err
}

func createBackoff(ctx context.Context, maxElapsedTime time.Duration) backoff.BackOffContext {
	exp := backoff.NewExponentialBackOff()
	exp.MaxElapsedTime = maxElapsedTime
	return backoff.WithContext(exp, ctx)
}
