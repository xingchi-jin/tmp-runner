package vault

type vaultSecret struct {
	Config *Config `json:"config"`
	Base64 bool    `json:"base64"`
	Path   string  `json:"path"`
	Key    string  `json:"key"`
}

type input struct {
	Secrets []*vaultSecret `json:"secrets"`
}

type VaultSecretTaskRequest struct {
	Action        string  `json:"action"`
	Config        *Config `json:"config"`
	EngineName    string  `json:"engine_name"`
	EngineVersion uint8   `json:"engine_version"`
	Key           string  `json:"key"`
	Path          string  `json:"path"`
	Value         string  `json:"value"`
}

type OperationStatus string

var (
	OperationStatusSuccess OperationStatus = "SUCCESS"
	OperationStatusFailure OperationStatus = "FAILURE"
)

type VaultSecretOperationResponse struct {
	Name            string          `json:"name"`
	Message         string          `json:"message"`
	Error           *Error          `json:"error"`
	OperationStatus OperationStatus `json:"status"`
}

type VaultSecretResponse struct {
	Value string `json:"value"`
	Error *Error `json:"error"`
}

type ValidationResponse struct {
	IsValid bool   `json:"valid"`
	Error   *Error `json:"error"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
	Status  int    `json:"status"`
}

// SecretResponse for fetch secret tasks

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
}
