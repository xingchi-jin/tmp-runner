package server

import (
	"github.com/harness/runner/delegateshell"
)

// System stores high level System sub-routines.
type System struct {
	delegate *delegateshell.DelegateShell
}

func NewSystem(delegate *delegateshell.DelegateShell) *System {
	return &System{
		delegate: delegate,
	}
}
