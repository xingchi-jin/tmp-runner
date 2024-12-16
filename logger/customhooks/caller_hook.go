package customhooks

import (
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var once sync.Once
var logrusPackage string

const (
	maximumCallerDepth = 25
	minimumCallerDepth = 4
)

// CorrectCallerHook adjusts the caller information to always point to the actual caller
type CorrectCallerHook struct{}

func (hook *CorrectCallerHook) Fire(entry *logrus.Entry) error {
	frame := getCaller(1)
	if frame != nil {
		entry.Caller = frame
	}
	return nil
}

func (hook *CorrectCallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// getCaller will return the caller after bypassing logrus calls and then skipping x frames
func getCaller(skip int) *runtime.Frame {
	once.Do(func() {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)
		for i := 0; i < maximumCallerDepth; i++ {
			funcName := runtime.FuncForPC(pcs[i]).Name()
			if strings.Contains(funcName, "fireHooks") {
				logrusPackage = getPackageName(funcName)
				break
			}
		}
	})

	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)
		if pkg != logrusPackage {
			if skip > 0 {
				skip--
				continue
			}
			return &f
		}
	}
	return nil
}

func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}
	return f
}
