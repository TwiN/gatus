package logging

import (
	"errors"
	"log/slog"
	"os"
	"time"
)

const (
	GatusLogSourceEnvVar     = "GATUS_LOG_SOURCE"
	GatusConfigLogTypeEnvVar = "GATUS_LOG_TYPE"
	GatusLogLevelEnvVar      = "GATUS_LOG_LEVEL"
)

var (
	ErrInvalidLevelString = errors.New("invalid log level string, must be one of: DEBUG, INFO, WARN, ERROR, FATAL")
	logLevels             = map[string]slog.Level{
		"DEBUG": slog.LevelDebug,
		"INFO":  slog.LevelInfo,
		"WARN":  slog.LevelWarn,
		"ERROR": slog.LevelError,
		"FATAL": slog.LevelError, // TODO in v6.0.0: Remove FATAL level support
	}

	logLevel = new(slog.LevelVar)
)

func Level() slog.Level {
	return logLevel.Level()
}

func levelFromString(level string) (slog.Level, error) {
	if slogLevel, exists := logLevels[level]; exists {
		return slogLevel, nil
	}
	return slog.LevelDebug, ErrInvalidLevelString
}

func Configure() {
	logHandlerOptions := &slog.HandlerOptions{Level: logLevel, AddSource: false}

	logSourceAsString := os.Getenv(GatusLogSourceEnvVar)
	switch logSourceAsString {
	case "", "FALSE":
		break
	case "TRUE":
		logHandlerOptions.AddSource = true
	default:
		slog.Warn("Invalid log source value, defaulting to false", "provided", logSourceAsString)
	}

	logTypeAsString := os.Getenv(GatusConfigLogTypeEnvVar)
	switch logTypeAsString {
	case "", "TEXT":
		logHandlerOptions.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "time" {
				a.Value = slog.StringValue(a.Value.Time().Format("2006-01-02|15:04:05"))
			}
			if a.Value.Kind() == slog.KindDuration {
				a.Value = slog.DurationValue(a.Value.Duration().Round(time.Millisecond))
			}
			return a
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, logHandlerOptions)))
		slog.Info("Log type set", "type", logTypeAsString)
	case "JSON":
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, logHandlerOptions)))
		slog.Info("Log type set", "type", logTypeAsString)
	default:
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, logHandlerOptions)))
		slog.Warn("Invalid log type", "provided", logTypeAsString, "default", "TEXT")
	}

	logLevelAsString := os.Getenv(GatusLogLevelEnvVar)
	if slogLevel, err := levelFromString(logLevelAsString); err != nil {
		logLevel.Set(slog.LevelInfo)
		if len(logLevelAsString) == 0 {
			slog.Info("Defaulting log level", "level", slog.LevelInfo)
		} else {
			slog.Warn("Invalid log level", "provided", logLevelAsString, "default", slog.LevelInfo)
		}
	} else {
		if logLevelAsString == "FATAL" {
			slog.Warn("FATAL log level deprecated, using ERROR level instead")
		}
		logLevel.Set(slogLevel)
		slog.Info("Log level set", "level", slogLevel)
	}
}
