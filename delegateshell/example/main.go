package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
	"github.com/harness/runner/tasks"
)

func main() {
	// Create a delegate client
	managerClient := client.New("https://localhost:9090", "kmpySmUISimoRrJL6NL73w" /* account id */, "2f6b0988b6fb3370073c3d0505baee59", true, "")

	// The poller needs a client that interacts with the task management system and a router to route the tasks
	keepAlive := heartbeat.New("kmpySmUISimoRrJL6NL73w", "2f6b0988b6fb3370073c3d0505baee59", "runner", []string{"macos-arm64"}, managerClient)

	// Register the poller
	ctx := context.Background()
	info, _ := keepAlive.Register(ctx)

	logrus.Info("Runner registered")

	requestsChan := make(chan *client.RunnerRequest, 3)

	// Start polling for bijou events
	eventsServer := poller.New(managerClient, requestsChan)
	// TODO: we don't need hb if we poll for task. Isn't it ? : )
	eventsServer.PollRunnerEvents(ctx, 3, info.ID, time.Second*10)

	logrus.Info("Read from chan")
	go func() {
		for {
			select {
			case req := <-requestsChan:
				logrus.Info("new task request")
				fmt.Println("task type: ", req.Task.Type)
				if req.Task.Type == "local_init" {
					logrus.Info("local init task")
					// unmarshal req.Task.Data into tasks.SetupRequest
					var setupRequest tasks.SetupRequest
					err := json.Unmarshal(req.Task.Data, &setupRequest)
					if err != nil {
						logrus.Error("Error occurred during unmarshalling. %w", err)
					}
					err = tasks.HandleSetup(ctx, setupRequest)
					if err != nil {
						logrus.Error("could not handle setup request: %w", err)
					}
				}
				logrus.Info(string(req.Task.Data))
			case <-ctx.Done():
				logrus.Info("exit")
			}
		}
	}()

	// Just to keep it running
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info(w, *info)
	})

	logrus.Info("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logrus.Fatal(err)
	}
}
