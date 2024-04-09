package tasks

import (
	"context"
	"encoding/json"
	"fmt"

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
	PipelineConfig     spec.PipelineConfig `json:"pipeline_config"`
	api.DestroyRequest `json:"destroy_request"`
}

// sampleDestroyRequest(id) creates a DestroyRequest object with the given id.
func SampleDestroyRequest(stageID string) DestroyRequest {
	return DestroyRequest{
		DestroyRequest: api.DestroyRequest{
			StageRuntimeID: stageID,
		},
		PipelineConfig: spec.PipelineConfig{
			Network: spec.Network{
				ID: sanitize(stageID),
			},
		},
	}
}

func HandleDestroy(ctx context.Context, s DestroyRequest) (api.VMTaskExecutionResponse, error) {
	err := engine.DestroyPipeline(ctx, engine.Opts{}, &s.PipelineConfig, internalStageLabel, s.StageRuntimeID)
	if err != nil {
		return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Failure, ErrorMessage: err.Error()}, nil
	}
	return api.VMTaskExecutionResponse{CommandExecutionStatus: api.Success}, nil
}
