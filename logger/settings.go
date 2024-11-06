package logger

import (
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	LogFileName = "runner.log"
)

func ConfigureLogging() {
	SetReportCaller(true)
	SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := path.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
			return "", fileName
		},
	})
}
