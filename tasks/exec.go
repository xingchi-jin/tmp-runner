package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine/spec"
	run "github.com/harness/lite-engine/pipeline/runtime"
	"github.com/sirupsen/logrus"
)

var (
	// this label is used to identify steps associated with a pipeline
	// It's used only internally to successfully destroy containers.
	internalStageLabel = "internal_stage_label"
)

func ExecHandler(ctx context.Context, req *task.Request) task.Response {
	// unmarshal req.Task.Data into tasks.SetupRequest
	var executeRequest ExecRequest
	err := json.Unmarshal(req.Task.Data, &executeRequest)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
	}
	fmt.Printf("execute request: %+v", executeRequest)
	resp, err := HandleExec(ctx, executeRequest)
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
	return task.Respond(respBytes)
}

type ExecRequest struct {
	// PipelineConfig is optional pipeline-level configuration which will be
	// used for step execution if specified.
	PipelineConfig  spec.PipelineConfig `json:"pipeline_config"`
	ExecStepRequest `json:"exec_request"`
}

type ExecStepRequest struct {
	api.StartStepRequest `json:"start_step_request"`
	StageRuntimeID       string `json:"stage_runtime_id"`
}

// sampleExecRequest(id) creates a ExecRequest object with the given id.
// It sets the network as the same ID (stage runtime ID which is unique)
// If image is empty, we use Host
func SampleExecRequest(stepID, stageID string, command []string, image string, entrypoint []string) ExecRequest {
	return ExecRequest{
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
		ExecStepRequest: ExecStepRequest{
			StartStepRequest: api.StartStepRequest{
				ID:             stepID,
				StageRuntimeID: stageID,
				LogConfig:      api.LogConfig{},
				TIConfig:       api.TIConfig{}, // only needed for a RunTest step
				Name:           "exec",
				WorkingDir:     generatePath(stageID),
				Kind:           api.Run,
				Network:        sanitize(stageID),
				Image:          image,
				Run: api.RunConfig{
					Command:    command,
					Entrypoint: entrypoint,
				},
				Volumes: []*spec.VolumeMount{
					{
						Name: "harness",
						Path: generatePath(stageID),
					},
				},
			}},
	}
}

func HandleExec(ctx context.Context, s ExecRequest) (api.VMTaskExecutionResponse, error) {
	fmt.Printf("in exec request, id is: %+v\n", s)
	if s.MountDockerSocket == nil || *s.MountDockerSocket { // required to support m1 where docker isn't installed.
		s.Volumes = append(s.Volumes, getDockerSockVolumeMount())
	}
	// Temporary hack - need to move this to java
	s.PipelineConfig.Volumes = append(
		s.PipelineConfig.Volumes,
		&spec.Volume{HostPath: &spec.VolumeHostPath{ID: "harness", Name: "harness", Path: s.WorkingDir, Create: true, Remove: true}})
	stepExecutor := run.NewStepExecutorStateless()
	// Internal label to keep track of containers started by a stage
	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}
	s.Labels[internalStageLabel] = s.StageRuntimeID
	resp, err := stepExecutor.Run(ctx, &s.StartStepRequest, &s.PipelineConfig)
	if err != nil {
		return api.VMTaskExecutionResponse{}, err
	}
	return resp, nil
}
