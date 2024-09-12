// Copyright 2024 Harness Inc. All rights reserved.
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
	"net/url"

	"github.com/harness/runner/utils"
)

const (
	endpoint = "%s/tasks"
)

type Client struct {
	utils.HTTPClient
}

func NewClient(baseUrl string) *Client {
	return &Client{
		HTTPClient: *utils.New(baseUrl, true, ""),
	}
}

// GetTasks retrieves the list of tasks running in a daemon set, specified by the `port` argument
func (p *Client) GetTasks(ctx context.Context, path string) (*DaemonTasks, error) {
	resp := &DaemonTasks{}
	fullpath := fmt.Sprintf(endpoint, path)
	_, err := p.doJson(ctx, fullpath, "GET", nil, resp)
	return resp, err
}

// Assign sends a new daemon task to a daemon set specified by the `port` argument
func (p *Client) Assign(ctx context.Context, path string, r *DaemonTasks) (*DaemonSetResponse, error) {
	req := r
	resp := &DaemonSetResponse{}
	fullpath := fmt.Sprintf(endpoint, path)
	_, err := p.doJson(ctx, fullpath, "POST", req, resp)
	return resp, err
}

// Remove removes a daemon task from a daemon set, specified by the `port` argument
func (p *Client) Remove(ctx context.Context, path string, r *[]string) (*DaemonSetResponse, error) {
	resp := &DaemonSetResponse{}
	fullpath := fmt.Sprintf(endpoint, path)

	// Build the query params
	queryParams := url.Values{}
	for _, id := range *r {
		queryParams.Add("taskIds", id)
	}

	// Append query parameters to the path
	fullpathWithQuery := fmt.Sprintf("%s?%s", fullpath, queryParams.Encode())
	_, err := p.doJson(ctx, fullpathWithQuery, "DELETE", nil, resp)
	return resp, err
}

func (p *Client) doJson(ctx context.Context, path, method string, in, out interface{}) (*http.Response, error) {
	var buf = &bytes.Buffer{}
	// marshal the input payload into json format and copy
	// to an io.ReadCloser.
	if in != nil {
		if err := json.NewEncoder(buf).Encode(in); err != nil {
			p.Logger.Errorf("could not encode input payload: %s", err)
		}
	}
	headers := make(map[string]string)
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
