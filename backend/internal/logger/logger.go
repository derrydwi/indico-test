// Package logger provides structured logging utilities
package logger

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
}

// ContextKey represents keys for context values
type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	UserIDKey    ContextKey = "user_id"
	TraceIDKey   ContextKey = "trace_id"
)

// New creates a new logger instance
func New(level, format string) *Logger {
	log := logrus.New()

	// Set log level
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)

	// Set formatter
	if format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	log.SetOutput(os.Stdout)

	return &Logger{Logger: log}
}

// WithContext adds context values to log fields
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.Logger.WithFields(logrus.Fields{})

	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}

	if userID := ctx.Value(UserIDKey); userID != nil {
		entry = entry.WithField("user_id", userID)
	}

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		entry = entry.WithField("trace_id", traceID)
	}

	return entry
}

// WithRequest adds request-specific fields
func (l *Logger) WithRequest(requestID, method, path string) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"method":     method,
		"path":       path,
	})
}

// WithError adds error information to log fields
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// WithJobID adds job ID to log fields
func (l *Logger) WithJobID(jobID string) *logrus.Entry {
	return l.Logger.WithField("job_id", jobID)
}

// WithComponent adds component name to log fields
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.Logger.WithField("component", component)
}

// Global logger instance
var globalLogger *Logger

// Init initializes the global logger
func Init(level, format string) {
	globalLogger = New(level, format)
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		globalLogger = New("info", "json")
	}
	return globalLogger
}

// Convenience functions for global logger
func WithContext(ctx context.Context) *logrus.Entry {
	return GetLogger().WithContext(ctx)
}

func WithRequest(requestID, method, path string) *logrus.Entry {
	return GetLogger().WithRequest(requestID, method, path)
}

func WithError(err error) *logrus.Entry {
	return GetLogger().WithError(err)
}

func WithJobID(jobID string) *logrus.Entry {
	return GetLogger().WithJobID(jobID)
}

func WithComponent(component string) *logrus.Entry {
	return GetLogger().WithComponent(component)
}

func Info(args ...interface{}) {
	GetLogger().Logger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	GetLogger().Logger.Infof(format, args...)
}

func Warn(args ...interface{}) {
	GetLogger().Logger.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	GetLogger().Logger.Warnf(format, args...)
}

func Error(args ...interface{}) {
	GetLogger().Logger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	GetLogger().Logger.Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	GetLogger().Logger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	GetLogger().Logger.Fatalf(format, args...)
}

func Debug(args ...interface{}) {
	GetLogger().Logger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	GetLogger().Logger.Debugf(format, args...)
}
