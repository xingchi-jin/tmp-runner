package tasks

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"unicode"

	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine"
	"github.com/harness/lite-engine/engine/spec"
)

type SetupRequest struct {
	ID               string `json:"id"` // stage runtime ID
	LogKey           string `json:"log_key"`
	api.SetupRequest `json:"setup_request"`
}

// exampleSetupRequest(id) creates a Request object with the given id.
// It sets the network as the same ID (stage runtime ID which is unique)
func exampleSetupRequest(id string) SetupRequest {
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

func HandleSetup(ctx context.Context, s SetupRequest) error {
	fmt.Printf("setup request: %+v", s)
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
		return err
	}
	return nil
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
