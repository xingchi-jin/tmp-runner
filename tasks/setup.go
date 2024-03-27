package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"unicode"

	runner_tasks "github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine"
	"github.com/harness/lite-engine/engine/spec"
	"github.com/sirupsen/logrus"
)

type SetupHandler struct{}

func (h *SetupHandler) Handle(ctx context.Context, req *runner_tasks.Request) runner_tasks.Response {
	var setupRequest SetupRequest
	err := json.Unmarshal(req.Task.Data, &setupRequest)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
		return runner_tasks.Error(err)
	}
	// TODO: remove this after delegate id no longer needed from setup request
	delegate_id := ctx.Value("delegate_id").(string)
	resp, err := HandleSetup(ctx, setupRequest, delegate_id)
	if err != nil {
		logrus.Error("could not handle setup request: %w", err)
		return runner_tasks.Error(err)
	}
	return runner_tasks.Respond(resp)
}

type SetupRequest struct {
	ID               string `json:"id"` // stage runtime ID
	LogKey           string `json:"log_key"`
	api.SetupRequest `json:"setup_request"`
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
func SampleSetupRequest(id string) SetupRequest {
	fmt.Printf("in setup request, id is: %s", id)
	return SetupRequest{
		ID: id,
		SetupRequest: api.SetupRequest{
			Network: spec.Network{
				ID: sanitize(id),
			},
			Volumes: []*spec.Volume{
				{
					HostPath: &spec.VolumeHostPath{
						ID:     "harness",
						Path:   generatePath(id),
						Create: true,
						Remove: true,
					},
				},
			},
		},
	}
}

func generatePath(id string) string {
	return fmt.Sprintf("/tmp/harness/%s", sanitize(id))
}

func sanitize(id string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return '_'
	}, id)
}

// TODO: Need to cleanup delegateID from here. Today, it's being used to route
// the subsequent tasks to the same delegate.
func HandleSetup(ctx context.Context, s SetupRequest, delegateID string) (SetupResponse, error) {
	fmt.Printf("setup request: %+v\n", s)
	if s.MountDockerSocket == nil || *s.MountDockerSocket { // required to support m1 where docker isn't installed.
		s.Volumes = append(s.Volumes, getDockerSockVolume())
	}
	cfg := &spec.PipelineConfig{
		Envs:    s.Envs,
		Network: s.Network,
		Platform: spec.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		Volumes:           s.Volumes,
		Files:             s.Files,
		EnableDockerSetup: s.MountDockerSocket,
		TTY:               s.TTY,
	}
	if err := engine.SetupPipeline(ctx, engine.Opts{}, cfg); err != nil {
		return SetupResponse{
			InfraType:        "DOCKER",
			DelegateMetaInfo: DelegateMetaInfo{ID: delegateID},
			VMTaskExecutionResponse: api.VMTaskExecutionResponse{
				CommandExecutionStatus: api.Failure,
				ErrorMessage:           err.Error()}}, nil
	}
	return SetupResponse{
		IPAddress:        "127.0.0.1",
		InfraType:        "DOCKER",
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
