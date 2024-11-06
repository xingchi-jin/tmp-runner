package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine/spec"
	"github.com/harness/runner/tasks/local"
	runnerspec "github.com/harness/runner/tasks/local/spec"
)

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

// SampleExecRequest creates a ExecRequest object with the given step and stage ID.
// It sets the network as the same ID (stage runtime ID which is unique)
// If image is empty, we assume the step is run on the host.
func SampleExecRequest(stepID, stageID string, command []string, image string, entrypoint []string) local.ExecRequest {
	return local.ExecRequest{
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
			Files: []*spec.File{&spec.File{
				Path:  "/tmp/abcd",
				Data:  "helloworld",
				Mode:  0400,
				IsDir: false,
			}},
			Network: sanitize(stageID),
			Image:   image,
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

// SampleSetupRequest creates a Request object with the given id.
// It sets the network as the same ID (stage runtime ID which is unique)
func SampleSetupRequest(stageID string) local.SetupRequest {
	return local.SetupRequest{
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

// SampleDestroyRequest(id) creates a DestroyRequest object with the given id.
func SampleDestroyRequest(stageID string) local.DestroyRequest {
	return local.DestroyRequest{
		Network: stageID,
		GroupID: stageID,
		Volumes: []*runnerspec.Volume{
			{
				HostPath: &runnerspec.VolumeHostPath{
					Path: generatePath(stageID),
					ID:   sanitize(stageID),
					Name: sanitize(stageID),
				},
			},
		},
	}
}
