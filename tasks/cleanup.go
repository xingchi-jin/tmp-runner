package tasks

import (
	"context"
	"encoding/json"
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
	Network string         `json:"network"`
	Volumes []*spec.Volume `json:"volumes"`
	// (optional) to delete containers, etc created using the group ID. Could be used in the future to delete other
	// resources created as part of the group.
	GroupID string `json:"group_id"`
}

func (d *DestroyRequest) Sanitize() {
	d.Network = sanitize(d.Network)
	d.GroupID = sanitize(d.GroupID)
}

// sampleDestroyRequest(id) creates a DestroyRequest object with the given id.
func SampleDestroyRequest(stageID string) DestroyRequest {
	return DestroyRequest{
		Network: stageID,
		GroupID: stageID,
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
		ctx, engine.Opts{}, pipelineConfig, internalStageLabel, s.GroupID)
	if err != nil {
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success}, nil
}
