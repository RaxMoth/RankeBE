package logger

import (
	"log/slog"
	"os"
	"strings"
)

var logger *slog.Logger

func Init(level string) {
	var logLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger = slog.New(handler)
}

func Debug(msg string, args ...interface{}) {
	if logger == nil {
		Init("info")
	}
	logger.Debug(msg, convertArgs(args)...)
}

func Info(msg string, args ...interface{}) {
	if logger == nil {
		Init("info")
	}
	logger.Info(msg, convertArgs(args)...)
}

func Warn(msg string, args ...interface{}) {
	if logger == nil {
		Init("info")
	}
	logger.Warn(msg, convertArgs(args)...)
}

func Error(msg string, args ...interface{}) {
	if logger == nil {
		Init("info")
	}
	logger.Error(msg, convertArgs(args)...)
}

func Fatal(msg string, args ...interface{}) {
	if logger == nil {
		Init("info")
	}
	logger.Error(msg, convertArgs(args)...)
	os.Exit(1)
}

func convertArgs(args []interface{}) []any {
	if len(args) == 1 {
		if fields, ok := args[0].(map[string]interface{}); ok {
			converted := make([]any, 0, len(fields)*2)
			for k, v := range fields {
				converted = append(converted, k, v)
			}
			return converted
		}
	}
	return args
}
