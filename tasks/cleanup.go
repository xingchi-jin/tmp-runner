package tasks

import (
	"context"
	"fmt"

	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine"
	"github.com/harness/lite-engine/engine/spec"
)

type destroyRequest struct {
	PipelineConfig spec.PipelineConfig `json:"pipeline_config"`
	api.DestroyRequest
}

// sampleDestroyRequest(id) creates a DestroyRequest object with the given id.
func sampleDestroyRequest(stageID string) destroyRequest {
	fmt.Printf("in destroy request, id is: %s", stageID)
	return destroyRequest{
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

func HandleDestroy(ctx context.Context, s destroyRequest) error {
	return engine.DestroyPipeline(ctx, engine.Opts{}, &s.PipelineConfig, internalStageLabel, s.StageRuntimeID)
}
