package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine/spec"
	"github.com/harness/lite-engine/logstream"
	run "github.com/harness/lite-engine/pipeline/runtime"
	"github.com/harness/runner/logger"
	"github.com/sirupsen/logrus"
)

var (
	// this label is used to identify steps associated with a pipeline
	// It's used only internally to successfully destroy containers.
	internalStageLabel = "internal_stage_label"
)

func ExecHandler(ctx context.Context, req *task.Request) task.Response {
	// unmarshal req.Task.Data into tasks.SetupRequest
	executeRequest := new(ExecRequest)
	err := json.Unmarshal(req.Task.Data, executeRequest)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
	}
	// Wrap the io.Writer to convert it into a logstream.Writer which is used by the lite-engine.
	resp, err := HandleExec(ctx, executeRequest, logger.NewWriterWrapper(req.Logger))
	if err != nil {
		logrus.Error("could not handle exec request: %w", err)
		panic(err)
	}
	// convert resp to bytes
	respBytes, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}
	fmt.Printf("exec response: %+v", resp)
	return task.Respond(respBytes)
}

func (s *ExecRequest) Sanitize() {
	s.Network = sanitize(s.Network)
	s.GroupID = sanitize(s.GroupID)
	s.ID = sanitize(s.ID)
	// TODO: Sanitize volumes and volume paths depending on the operating system.
}

type ExecRequest struct {
	// The struct in the engine uses `volumes` for volume mounts, so this is a temporary
	// workaround to be able to re-use the same structs.
	VolumesActual []*spec.Volume `json:"volumes_actual"`
	// (optional): used to label created containers as part of a group so they can be cleaned up easily.
	GroupID string `json:"group_id"`
	api.StartStepRequest
}

// sampleExecRequest(id) creates a ExecRequest object with the given id.
// It sets the network as the same ID (stage runtime ID which is unique)
// If image is empty, we use Host
func SampleExecRequest(stepID, stageID string, command []string, image string, entrypoint []string) ExecRequest {
	return ExecRequest{
		GroupID: stageID,
		VolumesActual: []*spec.Volume{
			{
				HostPath: &spec.VolumeHostPath{
					Name: sanitize(stageID),
					Path: generatePath(stageID),
					ID:   sanitize(stageID),
				},
			},
		},
		StartStepRequest: api.StartStepRequest{
			ID:         stepID,
			Name:       "exec",
			WorkingDir: generatePath(stageID),
			Kind:       api.Run,
			Network:    sanitize(stageID),
			Image:      image,
			Run: api.RunConfig{
				Command:    command,
				Entrypoint: entrypoint,
			},
			Volumes: []*spec.VolumeMount{
				{
					Name: sanitize(stageID),
					Path: generatePath(stageID),
				},
			},
		},
	}
}

func HandleExec(ctx context.Context, s *ExecRequest, writer logstream.Writer) (api.VMTaskExecutionResponse, error) {
	s.Sanitize()
	if s.MountDockerSocket == nil || *s.MountDockerSocket { // required to support m1 where docker isn't installed.
		s.Volumes = append(s.Volumes, getDockerSockVolumeMount())
	}
	// Create a new StepExecutor
	stepExecutor := run.NewStepExecutorStateless()
	// Internal label to keep track of containers started by a stage
	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}
	s.Labels[internalStageLabel] = s.GroupID
	// Map ExecRequest into what lite engine can understand
	pipelineConfig := &spec.PipelineConfig{
		Envs: s.Envs,
		Network: spec.Network{
			ID: s.Network,
		},
		Volumes: s.VolumesActual,
		Platform: spec.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
	}
	fmt.Printf("exec request: %+v\n", s)
	resp, err := stepExecutor.Run(ctx, &s.StartStepRequest, pipelineConfig, writer)
	if err != nil {
		return api.VMTaskExecutionResponse{}, err
	}
	return resp, nil
}
