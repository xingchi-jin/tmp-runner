package tasks

import (
	"context"
	"fmt"
	"runtime"

	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine/spec"
	run "github.com/harness/lite-engine/pipeline/runtime"
)

var (
	// this label is used to identify steps associated with a pipeline
	// It's used only internally to successfully destroy containers.
	internalStageLabel = "internal_stage_label"
)

type execRequest struct {
	// PipelineConfig is optional pipeline-level configuration which will be
	// used for step execution if specified.
	PipelineConfig spec.PipelineConfig `json:"pipeline_config"`
	api.StartStepRequest
}

// sampleExecRequest(id) creates a ExecRequest object with the given id.
// It sets the network as the same ID (stage runtime ID which is unique)
func sampleExecRequest(stepID, stageID string, command []string) execRequest {
	fmt.Printf("in exec request, id is: %s", stepID)
	return execRequest{
		PipelineConfig: spec.PipelineConfig{
			// This can be used from the step directly as well.
			Network: spec.Network{
				ID: sanitize(stageID),
			},
			Platform: spec.Platform{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
			},
		},
		StartStepRequest: api.StartStepRequest{
			ID:             stepID,
			StageRuntimeID: stageID,
			LogConfig:      api.LogConfig{},
			TIConfig:       api.TIConfig{}, // only needed for a RunTest step
			Name:           "exec",
			WorkingDir:     generatePath(stageID),
			Kind:           api.Run,
			Network:        sanitize(stageID),
			Image:          "alpine",
			Run: api.RunConfig{
				Command: command,
			},
			Volumes: []*spec.VolumeMount{
				{
					Name: "harness",
					Path: generatePath(stageID),
				},
			},
		},
	}
}

func HandleExec(ctx context.Context, s execRequest) (*api.PollStepResponse, error) {
	if s.MountDockerSocket == nil || *s.MountDockerSocket { // required to support m1 where docker isn't installed.
		s.Volumes = append(s.Volumes, getDockerSockVolumeMount())
	}
	stepExecutor := run.NewStepExecutorStateless()
	// Internal label to keep track of containers started by a stage
	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}
	s.Labels[internalStageLabel] = s.StageRuntimeID
	resp, err := stepExecutor.Run(ctx, &s.StartStepRequest, &s.PipelineConfig)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
