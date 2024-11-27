package local

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine/spec"
	logger "github.com/harness/lite-engine/logstream"
	run "github.com/harness/lite-engine/pipeline/runtime"
	runnerLogger "github.com/harness/runner/logger"
	"github.com/harness/runner/logger/logstream"
	"github.com/harness/runner/tasks/local/utils"
)

func ExecHandler(ctx context.Context, req *task.Request) task.Response {
	// unmarshal req.Task.Data into tasks.SetupRequest
	executeRequest := new(ExecRequest)
	err := json.Unmarshal(req.Task.Data, executeRequest)
	if err != nil {
		runnerLogger.Error("Error occurred during unmarshalling. %w", err)
		return task.Error(err)
	}
	// Wrap the io.Writer to convert it into a logstream.Writer which is used by the lite-engine.
	logWriter := logstream.NewWriterWrapper(req.Logger)
	logWriter.Open()
	// no need to close logWriter here, because
	// lite-engine's stepExecutor takes care of calling `logWriter.Close()`
	resp, err := HandleExec(ctx, executeRequest, logWriter)
	if err != nil {
		runnerLogger.Error("could not handle exec request: %w", err)
		return task.Error(err)
	}
	// convert resp to bytes
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return task.Error(err)
	}
	runnerLogger.Printf("exec response: %+v", resp)
	return task.Respond(respBytes)
}

func (s *ExecRequest) Sanitize() {
	s.Network = utils.Sanitize(s.Network)
	s.GroupID = utils.Sanitize(s.GroupID)
	s.ID = utils.Sanitize(s.ID)
	// TODO: Sanitize volumes and volume paths depending on the operating system.
}

type ExecRequest struct {
	// The struct in the engine uses `volumes` for volume mounts, so this is a temporary
	// workaround to be able to re-use the same structs.
	VolumesActual []*spec.Volume `json:"volumes_actual"`
	// (optional): used to label created containers as part of a group so they can be cleaned up easily.
	//
	// Deprecated: It's not being used. Labels are being used to identify containers for clean up.
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
					Name: utils.Sanitize(stageID),
					Path: utils.GeneratePath(stageID),
					ID:   utils.Sanitize(stageID),
				},
			},
		},
		StartStepRequest: api.StartStepRequest{
			ID:         stepID,
			Name:       "exec",
			WorkingDir: utils.GeneratePath(stageID),
			Kind:       api.Run,
			Network:    utils.Sanitize(stageID),
			Image:      image,
			Labels:     map[string]string{"harness": stageID},
			Run: api.RunConfig{
				Command:    command,
				Entrypoint: entrypoint,
			},
			Volumes: []*spec.VolumeMount{
				{
					Name: utils.Sanitize(stageID),
					Path: utils.GeneratePath(stageID),
				},
			},
		},
	}
}

func HandleExec(ctx context.Context, s *ExecRequest, writer logger.Writer) (api.VMTaskExecutionResponse, error) {
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
	//s.Labels["internal_stage_label"] = s.GroupID
	// Map ExecRequest into what lite engine can understand

	// Mount docker sock volume
	s.VolumesActual = append(s.VolumesActual, getDockerSockVolumeActualMount())

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
	// TODO sanitize this to not print the script in logs, because it can contain secrets
	fmt.Printf("exec request: %+v\n", s)
	resp, err := stepExecutor.Run(ctx, &s.StartStepRequest, pipelineConfig, writer)
	if err != nil {
		return api.VMTaskExecutionResponse{}, err
	}
	return resp, nil
}
