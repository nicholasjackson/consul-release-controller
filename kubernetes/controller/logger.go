package controller

import (
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-hclog"
)

// SinkLogger wraps a hclog.Logger with the LogSink interface
type SinkLogger struct {
	log hclog.Logger
}

func newSinkLogger(l hclog.Logger) *SinkLogger {
	return &SinkLogger{l}
}

// Init receives optional information about the logr library for LogSink
// implementations that need it.
func (s *SinkLogger) Init(info logr.RuntimeInfo) {}

// Enabled tests whether this LogSink is enabled at the specified V-level.
// For example, commandline flags might be used to set the logging
// verbosity and disable some info logs.
func (s *SinkLogger) Enabled(level int) bool {
	return true
}

// Info logs a non-error message with the given key/value pairs as context.
// The level argument is provided for optional logging.  This method will
// only be called when Enabled(level) is true. See Logger.Info for more
// details.
func (s *SinkLogger) Info(level int, msg string, keysAndValues ...interface{}) {
	s.log.Debug(msg, keysAndValues...)
}

// Error logs an error, with the given message and key/value pairs as
// context.  See Logger.Error for more details.
func (s *SinkLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	keysAndValues = append(keysAndValues, "error")
	keysAndValues = append(keysAndValues, err)
	s.log.Error(msg, keysAndValues...)
}

// WithValues returns a new LogSink with additional key/value pairs.  See
// Logger.WithValues for more details.
func (s *SinkLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return newSinkLogger(s.log.With(keysAndValues...))
}

// WithName returns a new LogSink with the specified name appended.  See
// Logger.WithName for more details.
func (s *SinkLogger) WithName(name string) logr.LogSink {
	return newSinkLogger(s.log.Named(name))
}
