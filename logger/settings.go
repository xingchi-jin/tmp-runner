package logger

import (
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	gologger "github.com/drone/runner-go/logger"
	"github.com/harness/runner/logger/customhooks"
	"github.com/sirupsen/logrus"
)

const (
	LogFileName = "runner.log"
)

func ConfigureLogging(debug, trace bool) {
	gologger.Default = gologger.Logrus(
		logrus.NewEntry(
			logrus.StandardLogger(),
		),
	)
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if trace {
		logrus.SetLevel(logrus.TraceLevel)
	}
	// Adding hooks
	AddHook(&customhooks.CorrectCallerHook{})
	AddHook(&customhooks.ContextHook{})
	AddHook(&customhooks.UTCHook{})

	SetReportCaller(true)
	SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			return "", getCallerFilenameAndLine(frame)
		},
	})
}

// getCallerFilenameAndLine returns the filename with the line number
func getCallerFilenameAndLine(frame *runtime.Frame) string {
	return filepath.Base(path.Clean(frame.File)) + ":" + strconv.Itoa(frame.Line)
}
