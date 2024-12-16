package customhooks

import (
	"github.com/sirupsen/logrus"
)

// ContextHook is a custom Logrus hook that extracts fields from context
// and adds it to the logrus entry
type ContextHook struct{}

func (hook *ContextHook) Fire(entry *logrus.Entry) error {
	if entry.Context == nil {
		return nil // No context provided, skip
	}
	// Extract taskId from context
	if labels, ok := entry.Context.Value(LogLabelsKey).(map[string]interface{}); ok {
		if inlineLabels, ok := labels[string(InlineLabelsKey)].(map[string]string); ok {
			for key, value := range inlineLabels {
				entry.Data[key] = value
			}
		}
	}
	return nil
}

func (hook *ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
