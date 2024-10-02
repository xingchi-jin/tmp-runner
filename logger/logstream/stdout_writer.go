// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
package logstream

import (
	"os"

	"github.com/harness/lite-engine/logstream"
)

type stdoutWriter struct{}

// newStdoutWriter creates a logstream.Writer
// that writes to stdout
func NewStdoutWriter() logstream.Writer {
	return &stdoutWriter{}
}

func (*stdoutWriter) Start()       {}
func (*stdoutWriter) Open() error  { return nil }
func (*stdoutWriter) Close() error { return nil }
func (*stdoutWriter) Error() error { return nil }

func (w *stdoutWriter) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}
