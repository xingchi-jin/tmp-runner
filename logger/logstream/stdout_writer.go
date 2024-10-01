package logstream

import (
	"os"

	"github.com/harness/lite-engine/logstream"
)

type stdoutWriter struct{}

// newStdoutWriter creates a logstream.Writer
// that writes to stdout
func newStdoutWriter() logstream.Writer {
	return &stdoutWriter{}
}

func (*stdoutWriter) Start()       {}
func (*stdoutWriter) Open() error  { return nil }
func (*stdoutWriter) Close() error { return nil }
func (*stdoutWriter) Error() error { return nil }

func (w *stdoutWriter) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}
