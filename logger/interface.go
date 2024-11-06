package logger

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	logger                = logrus.StandardLogger()
	closableHooks         []ClosableHook
	contextUpdatableHooks []ContextUpdatableHook
)

// ClosableHook is an interface for hooks that require a Close operation.
type ClosableHook interface {
	logrus.Hook
	Close() error
}

type ContextUpdatableHook interface {
	logrus.Hook
	UpdateContext(context map[string]string)
}

func GetLogger() *logrus.Logger {
	return logger
}

func getLogger() *logrus.Logger {
	return logger
}

// Configuration Methods

// SetReportCaller sets the log formatter.
func SetReportCaller(reportCaller bool) {
	getLogger().SetReportCaller(reportCaller)
}

// SetFormatter sets the log formatter.
func SetFormatter(formatter logrus.Formatter) {
	getLogger().SetFormatter(formatter)
}

// SetOutput sets the output destination for the logs.
func SetOutput(output io.Writer) {
	getLogger().SetOutput(output)
}

// AddHook adds a hook to the logger and tracks it if it's a ClosableHook.
func AddHook(hook logrus.Hook) {
	getLogger().AddHook(hook)
	if ch, ok := hook.(ClosableHook); ok {
		closableHooks = append(closableHooks, ch)
	}
	if ch, ok := hook.(ContextUpdatableHook); ok {
		contextUpdatableHooks = append(contextUpdatableHooks, ch)
	}
}

// CloseHooks closes all closable hooks.
func CloseHooks() error {
	var errorList []string
	for _, hook := range closableHooks {
		if err := hook.Close(); err != nil {
			errorList = append(errorList, err.Error())
		}
	}

	// If errorList is not empty, join the errors into a single error message
	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList, "; "))
	}

	return nil
}

// UpdateContextInHooks updates all updatable hooks with extra fields.
func UpdateContextInHooks(context map[string]string) {
	for _, hook := range contextUpdatableHooks {
		hook.UpdateContext(context)
	}
	return
}

// Log Methods

func WithError(err error) *logrus.Entry                      { return getLogger().WithError(err) }
func WithContext(ctx context.Context) *logrus.Entry          { return getLogger().WithContext(ctx) }
func WithField(key string, value interface{}) *logrus.Entry  { return getLogger().WithField(key, value) }
func WithFields(fields map[string]interface{}) *logrus.Entry { return getLogger().WithFields(fields) }
func WithTime(t time.Time) *logrus.Entry                     { return getLogger().WithTime(t) }

func Trace(args ...interface{})   { getLogger().Trace(args...) }
func Debug(args ...interface{})   { getLogger().Debug(args...) }
func Print(args ...interface{})   { getLogger().Print(args...) }
func Info(args ...interface{})    { getLogger().Info(args...) }
func Warn(args ...interface{})    { getLogger().Warn(args...) }
func Warning(args ...interface{}) { getLogger().Warning(args...) }
func Error(args ...interface{})   { getLogger().Error(args...) }
func Panic(args ...interface{})   { getLogger().Panic(args...) }
func Fatal(args ...interface{})   { getLogger().Fatal(args...) }

func Tracef(format string, args ...interface{})   { getLogger().Tracef(format, args...) }
func Debugf(format string, args ...interface{})   { getLogger().Debugf(format, args...) }
func Printf(format string, args ...interface{})   { getLogger().Printf(format, args...) }
func Infof(format string, args ...interface{})    { getLogger().Infof(format, args...) }
func Warnf(format string, args ...interface{})    { getLogger().Warnf(format, args...) }
func Warningf(format string, args ...interface{}) { getLogger().Warningf(format, args...) }
func Errorf(format string, args ...interface{})   { getLogger().Errorf(format, args...) }
func Panicf(format string, args ...interface{})   { getLogger().Panicf(format, args...) }
func Fatalf(format string, args ...interface{})   { getLogger().Fatalf(format, args...) }

func Traceln(args ...interface{})   { getLogger().Traceln(args...) }
func Debugln(args ...interface{})   { getLogger().Debugln(args...) }
func Println(args ...interface{})   { getLogger().Println(args...) }
func Infoln(args ...interface{})    { getLogger().Infoln(args...) }
func Warnln(args ...interface{})    { getLogger().Warnln(args...) }
func Warningln(args ...interface{}) { getLogger().Warningln(args...) }
func Errorln(args ...interface{})   { getLogger().Errorln(args...) }
func Panicln(args ...interface{})   { getLogger().Panicln(args...) }
func Fatalln(args ...interface{})   { getLogger().Fatalln(args...) }
