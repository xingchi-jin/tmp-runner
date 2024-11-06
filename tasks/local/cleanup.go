package local

import (
	"context"
	"encoding/json"

	"github.com/harness/runner/logger"

	"github.com/drone/go-task/task"

	"github.com/harness/lite-engine/api"
	"github.com/harness/runner/tasks/local/spec"
	"github.com/harness/runner/tasks/local/utils"
)

func DestroyHandler(ctx context.Context, req *task.Request) task.Response {
	var destroyRequest DestroyRequest
	err := json.Unmarshal(req.Task.Data, &destroyRequest)
	if err != nil {
		logger.Error("Error occurred during unmarshalling. %w", err)
		return task.Error(err)
	}
	resp, err := HandleDestroy(ctx, destroyRequest)
	if err != nil {
		logger.Error("could not handle destroy request: %w", err)
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
	//
	// Deprecated: It's not being used. The Containers field is used to identify containers to be destroyed.
	GroupID    string     `json:"group_id"`
	Containers Containers `json:"containers"`
}

type Containers struct {
	Labels map[string]string `json:"labels,omitempty"`
	// IDs     []string            `json:"ids,omitempty"` // not implemented
}

func (d *DestroyRequest) Sanitize() {
	d.Network = utils.Sanitize(d.Network)
	d.GroupID = utils.Sanitize(d.GroupID)
}

// sampleDestroyRequest(id) creates a DestroyRequest object with the given id.
func SampleDestroyRequest(stageID string) DestroyRequest {
	return DestroyRequest{
		Network: stageID,
		GroupID: stageID,
		Containers: Containers{
			Labels: map[string]string{"harness": stageID},
		},
		Volumes: []*spec.Volume{
			{
				HostPath: &spec.VolumeHostPath{
					Path: utils.GeneratePath(stageID),
					ID:   utils.Sanitize(stageID),
					Name: utils.Sanitize(stageID),
				},
			},
		},
	}
}

func HandleDestroy(ctx context.Context, s DestroyRequest) (api.VMTaskExecutionResponse, error) {
	s.Sanitize()
	// Clean up local host path
	utils.CleanupHostPathVolumes(s.Volumes)

	docker, err := utils.GetDockerClient()
	if err != nil {
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}

	if err := docker.KillContainersByLabel(ctx, s.Containers.Labels); err != nil {
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}

	if err := docker.RemoveNetworks(ctx, []string{s.Network}); err != nil {
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success}, nil
}
