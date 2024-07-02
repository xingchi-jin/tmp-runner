package runnerlogs

import (
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

func SetLogrus() {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			fileName := path.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
			//return frame.Function, fileName
			return "", fileName
		},
	})
}

// TODO: implement the support to send logs to google stackdriver.
// func getOutput() io.Writer {

// }
