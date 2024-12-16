package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"

	"github.com/harness/runner/delegateshell/delegate"
	"github.com/harness/runner/logger"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine"
	"github.com/harness/lite-engine/engine/spec"
	"github.com/harness/runner/logger/logstream"
	"github.com/harness/runner/tasks/local/utils"
)

type SetupHandler struct {
	taskContext *delegate.TaskContext
}

func NewSetupHandler(taskContext *delegate.TaskContext) *SetupHandler {
	return &SetupHandler{
		taskContext: taskContext,
	}
}

func (h *SetupHandler) Handle(ctx context.Context, req *task.Request) task.Response {
	setupRequest := new(SetupRequest)
	err := json.Unmarshal(req.Task.Data, setupRequest)
	if err != nil {
		logger.Error(ctx, "Error occurred during unmarshalling. %w", err)
		return task.Error(err)
	}
	logWriter := logstream.NewWriterWrapper(req.Logger)
	logWriter.Open()
	var delegateID string
	if h.taskContext.DelegateId != nil {
		delegateID = *h.taskContext.DelegateId
	}
	// TODO: remove this after delegate id no longer needed from setup request
	resp, err := HandleSetup(ctx, setupRequest, delegateID, logWriter)
	logWriter.Close()
	if err != nil {
		logger.Error(ctx, "could not handle setup request: %w", err)
		return task.Error(err)
	}
	logger.Printf(ctx, "setup response: %+v", resp)
	return task.Respond(resp)
}

type SetupRequest struct {
	Network spec.Network      `json:"network"`
	Volumes []*spec.Volume    `json:"volumes"`
	Envs    map[string]string `json:"envs"`
}

func (s *SetupRequest) Sanitize() {
	s.Network.ID = utils.Sanitize(s.Network.ID)
	// TODO: Sanitize volumes and volume paths depending on the operating system.
}

type DelegateMetaInfo struct {
	ID string `json:"id"`
}

type SetupResponse struct {
	IPAddress        string           `json:"ip_address"`
	DelegateMetaInfo DelegateMetaInfo `json:"delegate_meta_info"`
	InfraType        string           `json:"infra_type"`
	api.VMTaskExecutionResponse
}

// exampleSetupRequest(id) creates a Request object with the given id.
// It sets the network as the same ID (stage runtime ID which is unique)
func SampleSetupRequest(ctx context.Context, stageID string) SetupRequest {
	logger.Printf(ctx, "in setup request, id is: %s", stageID)
	return SetupRequest{
		Network: spec.Network{
			ID: utils.Sanitize(stageID),
		},
		Volumes: []*spec.Volume{
			{
				HostPath: &spec.VolumeHostPath{
					Name:   utils.Sanitize(stageID),
					Path:   utils.GeneratePath(stageID),
					ID:     utils.Sanitize(stageID),
					Create: true,
				},
			},
		},
	}
}

// TODO: Need to cleanup delegateID from here. Today, it's being used to route
// the subsequent tasks to the same delegate.
func HandleSetup(ctx context.Context, s *SetupRequest, delegateID string, logWriter io.Writer) (SetupResponse, error) {
	logger.Printf(ctx, "setup request: %+v\n", s)
	s.Sanitize()
	s.Volumes = append(s.Volumes, getDockerSockVolume())
	cfg := &spec.PipelineConfig{
		Envs:    s.Envs,
		Network: s.Network,
		Platform: spec.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		Volumes: s.Volumes,
	}
	logWriter.Write([]byte("setting up pipeline\n"))
	if err := engine.SetupPipeline(ctx, engine.Opts{}, cfg); err != nil {
		logWriter.Write([]byte(fmt.Sprintf("failed to set up pipeline: %s\n", err)))
		return SetupResponse{
			DelegateMetaInfo: DelegateMetaInfo{ID: delegateID},
			VMTaskExecutionResponse: api.VMTaskExecutionResponse{
				CommandExecutionStatus: api.Failure,
				ErrorMessage:           err.Error()}}, nil
	}
	logWriter.Write([]byte("pipeline set up successfully\n"))
	return SetupResponse{
		IPAddress: "127.0.0.1",
		// TODO: feature of "route back to the same delegate" should be handled at Runner framework level.
		DelegateMetaInfo: DelegateMetaInfo{ID: delegateID},
		VMTaskExecutionResponse: api.VMTaskExecutionResponse{
			CommandExecutionStatus: api.Success}}, nil
}

func getDockerSockVolume() *spec.Volume {
	path := engine.DockerSockUnixPath
	if runtime.GOOS == "windows" {
		path = engine.DockerSockWinPath
	}
	return &spec.Volume{
		HostPath: &spec.VolumeHostPath{
			Name: engine.DockerSockVolName,
			Path: path,
			ID:   "docker",
		},
	}
}

func getDockerSockVolumeMount() *spec.VolumeMount {
	path := engine.DockerSockUnixPath
	if runtime.GOOS == "windows" {
		path = engine.DockerSockWinPath
	}
	return &spec.VolumeMount{
		Name: engine.DockerSockVolName,
		Path: path,
	}
}

func getDockerSockVolumeActualMount() *spec.Volume {
	path := engine.DockerSockUnixPath
	if runtime.GOOS == "windows" {
		path = engine.DockerSockWinPath
	}
	return &spec.Volume{
		HostPath: &spec.VolumeHostPath{
			Name: engine.DockerSockVolName,
			Path: path,
			ID:   "docker",
		},
	}
}
