package vm

import "github.com/harness/runner/tasks/local"

type ExecRequest struct {
	Request  local.ExecRequest `json:"request"` // reuse local exec request
	Forward  *Forward          `json:"forward"`
	Metadata Metadata          `json:"metadata"`
}

// Forward represents the information about the VM where the request needs
// to be forwarded.
type Forward struct {
	ID        string `json:"id"`
	IPAddress string `json:"ip_address"`
	Certs     *Certs `json:"certs"`
}

type Certs struct {
	Public  []byte `json:"public"`
	Private []byte `json:"private"`
	CA      []byte `json:"ca"`
}
