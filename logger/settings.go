package logger

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	gologger "github.com/drone/runner-go/logger"
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
	AddHook(&CorrectCallerHook{})
	SetReportCaller(true)
	SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			return "", getCallerRelativePath(frame)
		},
	})
}

// getCallerRelativePath returns the relative filepath of the caller
func getCallerRelativePath(frame *runtime.Frame) string {
	// Get the filename from the current frame
	fileName := path.Clean(frame.File) + ":" + strconv.Itoa(frame.Line)
	// Get the base path and find the relative path
	// if not found, return the absolute path
	basePath, err := os.Getwd()
	if err != nil {
		return fileName
	}
	relativePath, err := filepath.Rel(basePath, fileName)
	if err != nil {
		return fileName
	}
	return relativePath
}
