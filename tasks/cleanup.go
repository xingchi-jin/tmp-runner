package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine"
	"github.com/harness/lite-engine/engine/spec"
	"github.com/sirupsen/logrus"
)

func DestroyHandler(ctx context.Context, req *task.Request) task.Response {
	var destroyRequest DestroyRequest
	err := json.Unmarshal(req.Task.Data, &destroyRequest)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
	}
	fmt.Printf("destroy request: %+v", destroyRequest)
	resp, err := HandleDestroy(ctx, destroyRequest)
	if err != nil {
		logrus.Error("could not handle destroy request: %w", err)
		panic(err)
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	return task.Respond(respBytes)
}

type DestroyRequest struct {
	Network        string         `json:"network"`
	Volumes        []*spec.Volume `json:"volumes"`
	ContainerLabel string         `json:"container_label"`
}

func (d *DestroyRequest) Sanitize() {
	d.Network = sanitize(d.Network)
	d.ContainerLabel = sanitize(d.ContainerLabel)
}

// sampleDestroyRequest(id) creates a DestroyRequest object with the given id.
func SampleDestroyRequest(stageID string) DestroyRequest {
	return DestroyRequest{
		Network:        stageID,
		ContainerLabel: stageID,
		Volumes: []*spec.Volume{
			{
				HostPath: &spec.VolumeHostPath{
					Path:   generatePath(stageID),
					ID:     sanitize(stageID),
					Name:   sanitize(stageID),
					Remove: true,
				},
			},
		},
	}
}

func HandleDestroy(ctx context.Context, s DestroyRequest) (api.VMTaskExecutionResponse, error) {
	pipelineConfig := &spec.PipelineConfig{
		Network: spec.Network{
			ID: s.Network,
		},
		Volumes: s.Volumes,
		Platform: spec.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
	}
	err := engine.DestroyPipeline(
		ctx, engine.Opts{}, pipelineConfig, internalStageLabel, s.ContainerLabel)
	if err != nil {
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success}, nil
}
