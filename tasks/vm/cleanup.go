package vm

type CleanupType string

const (
	Detach CleanupType = "DETACH"
	Delete CleanupType = "DELETE"
)

type CleanupRequest struct {
	Metadata Metadata  `json:"metadata"`
	Instance *Instance `json:"instance"`
}

type Instance struct {
	ID                 string      `json:"id"`
	StorageCleanupType CleanupType `json:"storage_cleanup_type"`
}
