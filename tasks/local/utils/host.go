package utils

import (
	"os"

	"github.com/harness/runner/tasks/local/spec"
)

func CleanupHostPathVolumes(volumes []*spec.Volume) {
	for _, vol := range volumes {
		if vol == nil || vol.HostPath == nil {
			continue
		}

		// TODO: Add logging
		path := vol.HostPath.Path
		os.RemoveAll(path)
	}
}
