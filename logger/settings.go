package logger

import (
	"path"
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
	SetReportCaller(true)
	SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := path.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
			return "", fileName
		},
	})
}
