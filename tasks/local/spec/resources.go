package spec

type (
	// Volume that can be mounted by containers.
	Volume struct {
		HostPath *VolumeHostPath `json:"host,omitempty"`
	}

	// VolumeHostPath mounts a file or directory from the
	// host node's filesystem into your container.
	VolumeHostPath struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Path string `json:"path,omitempty"`
	}
)
