package vm

type CommandExecutionStatus string

const (
	Success      CommandExecutionStatus = "SUCCESS"
	Failure      CommandExecutionStatus = "FAILURE"
	RunningState CommandExecutionStatus = "RUNNING"
	Queued       CommandExecutionStatus = "QUEUED"
	Skipped      CommandExecutionStatus = "SKIPPED"
)

type (
	VMTaskExecutionResponse struct {
		ErrorMessage           string                 `json:"error_message,omitempty"`
		IPAddress              string                 `json:"ip_address,omitempty"`
		OutputVars             map[string]string      `json:"output_vars,omitempty"`
		ServiceStatuses        []VMServiceStatus      `json:"service_statuses,omitempty"`
		CommandExecutionStatus CommandExecutionStatus `json:"command_execution_status,omitempty"`
		DelegateMetaInfo       DelegateMetaInfo       `json:"delegate_meta_info,omitempty"`
		Artifact               []byte                 `json:"artifact,omitempty"`
		PoolDriverUsed         string                 `json:"pool_driver_used,omitempty"`
		Outputs                []*OutputV2            `json:"outputs,omitempty"`
		OptimizationState      string                 `json:"optimization_state,omitempty"`
		GitspacesPortMappings  map[int]int            `json:"gitspaces_port_mappings,omitempty"`
	}

	DelegateMetaInfo struct {
		ID       string `json:"id"`
		HostName string `json:"host_name"`
	}

	VMServiceStatus struct {
		ID           string `json:"identifier"`
		Name         string `json:"name"`
		Image        string `json:"image"`
		LogKey       string `json:"log_key"`
		Status       Status `json:"status"`
		ErrorMessage string `json:"error_message"`
	}

	OutputV2 struct {
		Key   string     `json:"key,omitempty"`
		Value string     `json:"value"`
		Type  OutputType `json:"type,omitempty"`
	}
)

type OutputType string

const (
	OutputTypeString OutputType = "STRING"
	OutputTypeSecret OutputType = "SECRET"
)

type Status string

const (
	Running Status = "RUNNING"
	Error   Status = "ERROR"
)
