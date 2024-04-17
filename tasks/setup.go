package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"unicode"

	"github.com/drone/go-task/task"
	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine"
	"github.com/harness/lite-engine/engine/spec"
	"github.com/sirupsen/logrus"
)

func SetupHandler(ctx context.Context, req *task.Request) task.Response {
	var setupRequest SetupRequest
	err := json.Unmarshal(req.Task.Data, &setupRequest)
	if err != nil {
		logrus.Error("Error occurred during unmarshalling. %w", err)
		return task.Error(err)
	}
	// TODO: remove this after delegate id no longer needed from setup request
	delegate_id := ctx.Value("delegate_id").(string)
	resp, err := HandleSetup(ctx, &setupRequest, delegate_id)
	if err != nil {
		logrus.Error("could not handle setup request: %w", err)
		return task.Error(err)
	}
	fmt.Printf("setup response: %+v", resp)
	return task.Respond(resp)
}

type SetupRequest struct {
	Network spec.Network      `json:"network"`
	Volumes []*spec.Volume    `json:"volumes"`
	Envs    map[string]string `json:"envs"`
}

func (s *SetupRequest) Sanitize() {
	s.Network.ID = sanitize(s.Network.ID)
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
func SampleSetupRequest(stageID string) SetupRequest {
	fmt.Printf("in setup request, id is: %s", stageID)
	return SetupRequest{
		Network: spec.Network{
			ID: sanitize(stageID),
		},
		Volumes: []*spec.Volume{
			{
				HostPath: &spec.VolumeHostPath{
					Name:   sanitize(stageID),
					Path:   generatePath(stageID),
					ID:     sanitize(stageID),
					Create: true,
				},
			},
		},
	}
}

func generatePath(id string) string {
	return fmt.Sprintf("/tmp/harness/%s", sanitize(id))
}

// A function to sanitize any string and make it compatible with docker
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
func HandleSetup(ctx context.Context, s *SetupRequest, delegateID string) (SetupResponse, error) {
	fmt.Printf("setup request: %+v\n", s)
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
	if err := engine.SetupPipeline(ctx, engine.Opts{}, cfg); err != nil {
		return SetupResponse{
			DelegateMetaInfo: DelegateMetaInfo{ID: delegateID},
			VMTaskExecutionResponse: api.VMTaskExecutionResponse{
				CommandExecutionStatus: api.Failure,
				ErrorMessage:           err.Error()}}, nil
	}
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
