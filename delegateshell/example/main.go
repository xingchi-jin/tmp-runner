package main

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
)

func main() {
	// Create a delegate client
	managerClient := client.New("https://localhost:9090", "kmpySmUISimoRrJL6NL73w" /* account id */, "delegate token", true, "")

	// The poller needs a client that interacts with the task management system and a router to route the tasks
	keepAlive := heartbeat.New("kmpySmUISimoRrJL6NL73w", "runner", []string{"macos-arm64"}, managerClient)

	// Register the poller
	ctx := context.Background()
	info, _ := keepAlive.Register(ctx)

	logrus.Info("Runner registered")

	// Start polling for task events
	eventsServer := poller.New(managerClient, NewRouter())
	eventsServer.PollRunnerEvents(ctx, 3, info.ID, 2*time.Second)
}
