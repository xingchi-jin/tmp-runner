package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/harness/lite-engine/api"
	"github.com/harness/runner/delegateshell/client"
	"github.com/harness/runner/delegateshell/heartbeat"
	"github.com/harness/runner/delegateshell/poller"
	"github.com/harness/runner/tasks"
)

func main() {
	// Create a delegate client
	managerClient := client.New("https://localhost:9090", "kmpySmUISimoRrJL6NL73w" /* account id */, "2f6b0988b6fb3370073c3d0505baee59", true, "")

	// The poller needs a client that interacts with the task management system and a router to route the tasks
	keepAlive := heartbeat.New("kmpySmUISimoRrJL6NL73w", "runner", []string{"macos-arm64"}, managerClient)

	// Register the poller
	ctx := context.Background()
	info, _ := keepAlive.Register(ctx)

	logrus.Info("Runner registered")

	requestsChan := make(chan *client.RunnerRequest, 3)

	// Start polling for bijou events
	eventsServer := poller.New(managerClient, requestsChan)
	// TODO: we don't need hb if we poll for task. Isn't it ? : )
	eventsServer.PollRunnerEvents(ctx, 3, info.ID, time.Second*10)

	// add a map which can store the taskID to prevent getting duplicates in the request channel
	m := make(map[string]bool)

	logrus.Info("Read from chan")
	go func() {
		for {
			select {
			case req := <-requestsChan:
				fmt.Println("task id: ", req.TaskId)
				if _, ok := m[req.TaskId]; ok {
					logrus.Info("task already exists")
					continue
				}
				m[req.TaskId] = true

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
						panic(err)
					}
					respBytes, err := json.Marshal(api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success})
					if err != nil {
						panic(err)
					}
					err = managerClient.SendStatus(ctx, info.ID, req.TaskId, &client.TaskResponse{Type: "INITIALIZATION_PHASE", Code: "OK", Data: respBytes})
					if err != nil {
						logrus.Error("could not return back status: %w", err)
						panic(err)
					}
				} else if req.Task.Type == "local_execute" {
					logrus.Info("local init task")

					// unmarshal req.Task.Data into tasks.SetupRequest
					var executeRequest tasks.ExecRequest
					err := json.Unmarshal(req.Task.Data, &executeRequest)
					if err != nil {
						logrus.Error("Error occurred during unmarshalling. %w", err)
					}
					fmt.Printf("execute request: %+v", executeRequest)
					resp, err := tasks.HandleExec(ctx, executeRequest)
					if err != nil {
						logrus.Error("could not handle setup request: %w", err)
						panic(err)
					}
					// convert resp to bytes
					respBytes, err := json.Marshal(resp)
					if err != nil {
						panic(err)
					}
					fmt.Println("info.ID: ")
					err = managerClient.SendStatus(ctx, info.ID, req.TaskId, &client.TaskResponse{Type: "CI_EXECUTE_STEP", Code: "OK", Data: respBytes})
					if err != nil {
						logrus.Error("could not return back status: %w", err)
						panic(err)
					}
				} else if req.Task.Type == "local_cleanup" {
					logrus.Info("local cleanup task")
					// unmarshal req.Task.Data into tasks.SetupRequest
					var destroyRequest tasks.DestroyRequest
					err := json.Unmarshal(req.Task.Data, &destroyRequest)
					if err != nil {
						logrus.Error("Error occurred during unmarshalling. %w", err)
					}
					fmt.Printf("destroy request: %+v", destroyRequest)
					err = tasks.HandleDestroy(ctx, destroyRequest)
					if err != nil {
						logrus.Error("could not handle destroy request: %w", err)
						panic(err)
					}
					respBytes, err := json.Marshal(api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success})
					if err != nil {
						panic(err)
					}
					err = managerClient.SendStatus(ctx, info.ID, req.TaskId, &client.TaskResponse{Type: "CI_CLEANUP", Code: "OK", Data: respBytes})
					if err != nil {
						logrus.Error("could not return back status: %w", err)
						panic(err)
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
