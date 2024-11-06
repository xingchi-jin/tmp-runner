package gcplogger

import (
	"context"

	"github.com/harness/runner/logger"
	"github.com/sirupsen/logrus"

	"cloud.google.com/go/logging"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// refer https://cloud.google.com/go/docs/reference/cloud.google.com/go/logging/latest
type gcpLoggingHook struct {
	client  *logging.Client
	logger  *logging.Logger
	context map[string]string
}

func newGcpLoggingHook(ctx context.Context, logID string, projectId string, tokenSource oauth2.TokenSource) (*gcpLoggingHook, error) {
	client, err := logging.NewClient(ctx, projectId, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	hook := &gcpLoggingHook{
		client:  client,
		logger:  client.Logger(logID),
		context: map[string]string{},
	}

	hook.client.OnError = hook.onError

	return hook, nil
}

func (hook *gcpLoggingHook) onError(err error) {
	logger.WithError(err).Error("Error detected from stack driver")
}

func (hook *gcpLoggingHook) Close() error {
	return hook.client.Close()
}

func (hook *gcpLoggingHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *gcpLoggingHook) Fire(entry *logrus.Entry) error {

	payload := map[string]interface{}{
		"message": entry.Message,
		"fields":  entry.Data,

		// Adds extra fields specifically for improved filtering in Stack driver logs.
		// This applies only to remote logging, keeping other log outputs uncluttered.
		"labels": hook.context,
	}

	if entry.HasCaller() {
		payload["reportLocation"] = map[string]interface{}{
			"filePath":     entry.Caller.File,
			"functionName": entry.Caller.Function,
			"lineNumber":   entry.Caller.Line,
		}
	}

	if errValue, ok := entry.Data[logrus.ErrorKey]; ok {
		if err, isErr := errValue.(error); isErr {
			payload["error"] = err.Error()
		}
	}

	severity := getSeverity(entry.Level)
	hook.logger.Log(logging.Entry{
		Payload:  payload,
		Severity: severity,
	})

	return nil
}

func getSeverity(level logrus.Level) logging.Severity {
	switch level {
	case logrus.DebugLevel:
		return logging.Debug
	case logrus.InfoLevel:
		return logging.Info
	case logrus.WarnLevel:
		return logging.Warning
	case logrus.ErrorLevel:
		return logging.Error
	case logrus.FatalLevel, logrus.PanicLevel:
		return logging.Critical
	default:
		return logging.Default
	}
}

func (hook *gcpLoggingHook) UpdateContext(context map[string]string) {
	for key, value := range context {
		hook.context[key] = value
	}
}
