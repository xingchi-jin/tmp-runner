package customhooks

import "github.com/sirupsen/logrus"

// UTCHook ensures that the log entry timestamps are in UTC
type UTCHook struct{}

func (h *UTCHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *UTCHook) Fire(entry *logrus.Entry) error {
	entry.Time = entry.Time.UTC()
	return nil
}
