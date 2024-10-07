package logstream

import (
	"io"

	"github.com/harness/lite-engine/logstream"
)

type wrapper struct {
	writer io.Writer
}

// NewWriterWrapper converts an io.Writer into a logstream.Writer
func NewWriterWrapper(writer io.Writer) logstream.Writer {
	return &wrapper{
		writer: writer,
	}
}

func (*wrapper) Start()       {}
func (*wrapper) Error() error { return nil }

func (w *wrapper) Open() error {
	logstreamWriter, ok := w.writer.(logstream.Writer)
	if ok {
		return logstreamWriter.Open()
	}
	return nil
}

func (w *wrapper) Close() error {
	logstreamWriter, ok := w.writer.(logstream.Writer)
	if ok {
		return logstreamWriter.Close()
	}
	return nil
}

func (w *wrapper) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}
