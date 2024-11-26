package vm

import "github.com/drone-runners/drone-runner-aws/command/harness"

// Metadata needed for alerting/metrics/logging purposes
type Metadata struct {
	AccountID      string `json:"account_id,omitempty"`
	OrgID          string `json:"org_id,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	PipelineID     string `json:"pipeline_id,omitempty"`
	RunSequence    int    `json:"run_sequence,omitempty"`
	StageRuntimeID string `json:"stage_runtime_id,omitempty"`
	IsFreeAccount  bool   `json:"is_free_account,omitempty"`
}

func convertMetadata(metadata Metadata) harness.Context {
	return harness.Context{
		AccountID:     metadata.AccountID,
		OrgID:         metadata.OrgID,
		ProjectID:     metadata.ProjectID,
		PipelineID:    metadata.PipelineID,
		RunSequence:   metadata.RunSequence,
		IsFreeAccount: metadata.IsFreeAccount,
	}
}
