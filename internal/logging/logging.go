package logging

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger is a wrapper around zerolog.Logger
type Logger struct {
	logger zerolog.Logger
}

// Config holds logging configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, console
	Output     string // stdout, stderr, file path
	TimeFormat string // RFC3339, RFC3339Nano, Unix, etc.
}

// NewLogger creates a new logger with the given configuration
func NewLogger(cfg Config) (*Logger, error) {
	var output io.Writer

	// Set output
	switch cfg.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// Assume it's a file path
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		output = file
	}

	// Set format
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}
	}

	// Set log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	// Create logger
	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	// Set global logger
	log.Logger = logger

	return &Logger{logger: logger}, nil
}

// WithContext adds context to the logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{logger: l.logger.With().Ctx(ctx).Logger()}
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{logger: l.logger.With().Interface(key, value).Logger()}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	logger := l.logger.With()
	for k, v := range fields {
		logger = logger.Interface(k, v)
	}
	return &Logger{logger: logger.Logger()}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}

// ErrorWithErr logs an error message with an error
func (l *Logger) ErrorWithErr(msg string, err error) {
	l.logger.Error().Err(err).Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.logger.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal().Msgf(format, args...)
}

// WithError adds an error to the logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{logger: l.logger.With().Err(err).Logger()}
}

// WithRequestID adds a request ID to the logger
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{logger: l.logger.With().Str("request_id", requestID).Logger()}
}

// WithJobID adds a job ID to the logger
func (l *Logger) WithJobID(jobID string) *Logger {
	return &Logger{logger: l.logger.With().Str("job_id", jobID).Logger()}
}

// WithVideoID adds a video ID to the logger
func (l *Logger) WithVideoID(videoID string) *Logger {
	return &Logger{logger: l.logger.With().Str("video_id", videoID).Logger()}
}

// WithWorkerID adds a worker ID to the logger
func (l *Logger) WithWorkerID(workerID string) *Logger {
	return &Logger{logger: l.logger.With().Str("worker_id", workerID).Logger()}
}

// LogHTTPRequest logs HTTP request details
func (l *Logger) LogHTTPRequest(method, path, clientIP string, statusCode int, duration time.Duration) {
	l.logger.Info().
		Str("method", method).
		Str("path", path).
		Str("client_ip", clientIP).
		Int("status_code", statusCode).
		Dur("duration_ms", duration).
		Msg("HTTP request")
}

// LogJobEvent logs a job-related event
func (l *Logger) LogJobEvent(jobID, event, status string, details map[string]interface{}) {
	evt := l.logger.Info().
		Str("job_id", jobID).
		Str("event", event).
		Str("status", status)

	for k, v := range details {
		evt = evt.Interface(k, v)
	}

	evt.Msg("Job event")
}

// LogTranscodingProgress logs transcoding progress
func (l *Logger) LogTranscodingProgress(jobID string, progress float64, fps float64, speed float64) {
	l.logger.Info().
		Str("job_id", jobID).
		Float64("progress", progress).
		Float64("fps", fps).
		Float64("speed", speed).
		Msg("Transcoding progress")
}

// LogStorageOperation logs a storage operation
func (l *Logger) LogStorageOperation(operation, bucket, key string, size int64, duration time.Duration, err error) {
	evt := l.logger.Info()
	if err != nil {
		evt = l.logger.Error().Err(err)
	}

	evt.
		Str("operation", operation).
		Str("bucket", bucket).
		Str("key", key).
		Int64("size_bytes", size).
		Dur("duration_ms", duration).
		Msg("Storage operation")
}

// LogDatabaseOperation logs a database operation
func (l *Logger) LogDatabaseOperation(operation string, duration time.Duration, err error) {
	evt := l.logger.Info()
	if err != nil {
		evt = l.logger.Error().Err(err)
	}

	evt.
		Str("operation", operation).
		Dur("duration_ms", duration).
		Msg("Database operation")
}

// LogGPUMetrics logs GPU metrics
func (l *Logger) LogGPUMetrics(gpuID string, utilization, memoryUsed, temperature float64) {
	l.logger.Info().
		Str("gpu_id", gpuID).
		Float64("utilization_percent", utilization).
		Float64("memory_used_mb", memoryUsed).
		Float64("temperature_celsius", temperature).
		Msg("GPU metrics")
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() (*Logger, error) {
	return NewLogger(Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		TimeFormat: time.RFC3339,
	})
}

// NewConsoleLogger creates a logger with console output for development
func NewConsoleLogger() (*Logger, error) {
	return NewLogger(Config{
		Level:      "debug",
		Format:     "console",
		Output:     "stdout",
		TimeFormat: time.RFC3339,
	})
}
