// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package heartbeat

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/harness/runner/logger"

	"github.com/harness/runner/delegateshell/client"
	"github.com/icrowley/fake"

	"github.com/pkg/errors"
)

var (
	// Time period between sending heartbeats to the server
	hearbeatInterval  = 10 * time.Second
	heartbeatTimeout  = 15 * time.Second
	taskEventsTimeout = 30 * time.Second
)

type FilterFn func(*client.TaskEvent) bool

type KeepAlive struct {
	AccountID string
	Name      string   // name of the runner
	Tags      []string // list of tags that the runner accepts
	Client    client.Client
	Filter    FilterFn
	// The Harness manager allows two task acquire calls with the same delegate ID to go through (by design).
	// We need to make sure two different threads do not acquire the same task.
	// This map makes sure Acquire() is called only once per task ID. The mapping is removed once the status
	// for the task has been sent.
	m sync.Map
}

type DelegateInfo struct {
	Host string
	IP   string
	ID   string
	Name string
}

func New(accountID, name string, tags []string, c client.Client) *KeepAlive {
	return &KeepAlive{
		AccountID: accountID,
		Tags:      tags,
		Name:      name,
		Client:    c,
		m:         sync.Map{},
	}
}

func (p *KeepAlive) SetFilter(filter FilterFn) {
	p.Filter = filter
}

// Register registers the runner with the server. The server generates a delegate ID
// which is returned to the client.
func (p *KeepAlive) Register(ctx context.Context) (*DelegateInfo, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrap(err, "could not get host name")
	}
	host = "runner-" + strings.ReplaceAll(host, " ", "-")
	ip := getOutboundIP()
	id, err := p.register(ctx, ip, host)
	if err != nil {
		logger.WithField("ip", ip).WithField("host", host).WithError(err).Error("could not register runner")
		return nil, err
	}
	return &DelegateInfo{
		ID:   id,
		Host: host,
		IP:   ip,
		Name: p.Name,
	}, nil
}

// Register registers the runner and runs a background thread which keeps pinging the server
// at a period of interval. It returns the delegate ID.
func (p *KeepAlive) register(ctx context.Context, ip, host string) (string, error) {
	req := p.getRegisterRequest("", ip, host)
	resp, err := p.Client.Register(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "could not register the runner")
	}
	req.ID = resp.Resource.DelegateID
	logger.WithField("id", req.ID).WithField("host", req.HostName).
		WithField("ip", req.IP).Info("registered delegate successfully")
	return resp.Resource.DelegateID, nil
}

// Heartbeat starts a periodic thread in the background which continually pings the server
func (p *KeepAlive) Heartbeat(ctx context.Context, id, ip, host string) {
	req := p.getRegisterRequest(id, ip, host)
	go func() {
		msgDelayTimer := time.NewTimer(hearbeatInterval)
		defer msgDelayTimer.Stop()
		for {
			msgDelayTimer.Reset(hearbeatInterval)
			select {
			case <-ctx.Done():
				logger.Infoln("context canceled, stopping heartbeat")
				return
			case <-msgDelayTimer.C:
				req.LastHeartbeat = time.Now().UnixMilli()
				heartbeatCtx, cancelFn := context.WithTimeout(ctx, heartbeatTimeout)
				err := p.Client.Heartbeat(heartbeatCtx, req)
				if err != nil && !errors.Is(err, context.Canceled) {
					logger.WithError(err).Errorf("could not send heartbeat")
				}
				cancelFn()
			}
		}
	}()
}

func (p *KeepAlive) getRegisterRequest(id, ip, host string) *client.RegisterRequest {
	req := &client.RegisterRequest{
		AccountID:     p.AccountID,
		RunnerName:    p.Name,
		LastHeartbeat: time.Now().UnixMilli(),
		//Token:              p.AccountSecret,
		NG:       true,
		Type:     "DOCKER",
		Polling:  true,
		HostName: host,
		IP:       ip,
		// SupportedTaskTypes: p.Router.Routes(),  // Ignore this because for new Runner tasks, this SupportedTaskTypes feature doesn't apply
		Tags:              p.Tags,
		Version:           "v0.1",
		HeartbeatAsObject: true,
	}

	if id != "" {
		req.ID = id
	}
	return req
}

// Get preferred outbound ip of this machine. It returns a fake IP in case of errors.
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		logger.WithError(err).Error("could not figure out an IP, using a randomly generated IP")
		return "fake-" + fake.IPv4()
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
