// Package logger provides structured logging for the Reservia API.
package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger defines the logging interface.
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// logrusLogger implements Logger using logrus.
type logrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// New creates a new logger instance.
func New() Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	return &logrusLogger{
		logger: logger,
		entry:  logger.WithFields(logrus.Fields{}),
	}
}

// Debug logs a debug message.
func (l *logrusLogger) Debug(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Debug(msg)
}

// Info logs an info message.
func (l *logrusLogger) Info(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Info(msg)
}

// Warn logs a warning message.
func (l *logrusLogger) Warn(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Warn(msg)
}

// Error logs an error message.
func (l *logrusLogger) Error(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Error(msg)
}

// Fatal logs a fatal message and exits.
func (l *logrusLogger) Fatal(msg string, fields ...interface{}) {
	l.entry.WithFields(parseFields(fields...)).Fatal(msg)
}

// WithField adds a field to the logger context.
func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return &logrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields adds multiple fields to the logger context.
func (l *logrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &logrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(fields),
	}
}

// parseFields converts variadic interface{} to logrus.Fields.
func parseFields(fields ...interface{}) logrus.Fields {
	logFields := logrus.Fields{}

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				logFields[key] = fields[i+1]
			}
		}
	}

	return logFields
}
