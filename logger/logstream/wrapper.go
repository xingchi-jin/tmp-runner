package logstream

import (
	"io"

	"github.com/harness/lite-engine/logstream"
)

type wrapper struct {
	writer io.Writer
}

// NewWriterWrapper converts an io.Writer into a logstream.Writer
// It mocks out the other functions and keeps the write intact.
func NewWriterWrapper(writer io.Writer) logstream.Writer {
	return &wrapper{
		writer: writer,
	}
}

func (*wrapper) Start()       {}
func (*wrapper) Open() error  { return nil }
func (*wrapper) Close() error { return nil }
func (*wrapper) Error() error { return nil }

func (w *wrapper) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}
