package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Level represents logging levels
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging
type Logger struct {
	level  Level
	logger *log.Logger
}

// Fields represents structured log fields
type Fields map[string]interface{}

var defaultLogger *Logger

func init() {
	defaultLogger = New(INFO)
}

// New creates a new logger with the specified level
func New(level Level) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

// NewFromString creates a logger from a level string
func NewFromString(levelStr string) *Logger {
	level := INFO
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = DEBUG
	case "INFO":
		level = INFO
	case "WARN":
		level = WARN
	case "ERROR":
		level = ERROR
	}
	return New(level)
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	defaultLogger = l
}

// logEntry represents a structured log entry
type logEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// log writes a log entry
func (l *Logger) log(level Level, msg string, fields Fields) {
	if level < l.level {
		return
	}

	entry := logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   msg,
		Fields:    fields,
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging
		l.logger.Printf("[%s] %s %v", level.String(), msg, fields)
		return
	}

	l.logger.Println(string(data))
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Fields) {
	l.log(DEBUG, msg, mergeFields(fields...))
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Fields) {
	l.log(INFO, msg, mergeFields(fields...))
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Fields) {
	l.log(WARN, msg, mergeFields(fields...))
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...Fields) {
	l.log(ERROR, msg, mergeFields(fields...))
}

// WithContext returns a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract common context values like request ID, trace ID, etc.
	// This is a placeholder for AWS Lambda context extraction
	return l
}

// mergeFields combines multiple field maps
func mergeFields(fields ...Fields) Fields {
	if len(fields) == 0 {
		return nil
	}
	result := make(Fields)
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

// Package-level convenience functions using the default logger

// Debug logs a debug message using the default logger
func Debug(msg string, fields ...Fields) {
	defaultLogger.Debug(msg, fields...)
}

// Info logs an info message using the default logger
func Info(msg string, fields ...Fields) {
	defaultLogger.Info(msg, fields...)
}

// Warn logs a warning message using the default logger
func Warn(msg string, fields ...Fields) {
	defaultLogger.Warn(msg, fields...)
}

// Error logs an error message using the default logger
func Error(msg string, fields ...Fields) {
	defaultLogger.Error(msg, fields...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	defaultLogger.Error(fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	defaultLogger.Info(fmt.Sprintf(format, args...))
}
