package logging

import (
	"errors"
	"log/slog"
	"os"
)

const (
	GatusLogSourceEnvVar = "GATUS_LOG_SOURCE"
	GatusLogTypeEnvVar   = "GATUS_LOG_TYPE"
	GatusLogLevelEnvVar  = "GATUS_LOG_LEVEL"

	DefaultLogType  = "TEXT"
	DefaultLogLevel = "INFO"
)

var (
	ErrInvalidLevelString = errors.New("invalid log level string, must be one of: DEBUG, INFO, WARN, ERROR, FATAL")
	logLevels             = map[string]slog.Level{
		"DEBUG": slog.LevelDebug,
		"INFO":  slog.LevelInfo,
		"WARN":  slog.LevelWarn,
		"ERROR": slog.LevelError,
		// XXX: Remove this in v6.0.0
		"FATAL": slog.LevelError,
		// XXX: End of v6.0.0 removals
	}

	logLevel slog.Level
)

func Level() slog.Level {
	return logLevel
}

func levelFromString(level string) (slog.Level, error) {
	if slogLevel, exists := logLevels[level]; exists {
		return slogLevel, nil
	}
	return logLevel, ErrInvalidLevelString
}

func getConfiguredLogLevel() slog.Level {
	levelAsString := os.Getenv(GatusLogLevelEnvVar)
	if len(levelAsString) == 0 {
		return logLevels[DefaultLogLevel]
	} else if level, err := levelFromString(levelAsString); err != nil {
		slog.Warn("Invalid log level, using default", "provided", level, "default", DefaultLogLevel)
		return logLevels[DefaultLogLevel]
	} else {
		// XXX: Remove this in v6.0.0
		if levelAsString == "FATAL" {
			slog.Warn("FATAL log level has been deprecated and will be removed in v6.0.0")
			slog.Warn("Please use the ERROR log level instead")
		}
		// XXX: End of v6.0.0 removals
		return level
	}
}

func getConfiguredLogSource() bool {
	logSourceAsString := os.Getenv(GatusLogSourceEnvVar)
	if len(logSourceAsString) == 0 {
		return false
	} else if logSourceAsString != "TRUE" && logSourceAsString != "FALSE" {
		slog.Warn("Invalid log source", "provided", logSourceAsString, "default", "FALSE")
		return false
	}
	return logSourceAsString == "TRUE"
}

func Configure() {
	logTypeAsString := os.Getenv(GatusLogTypeEnvVar)
	switch logTypeAsString {
	case "", "TEXT":
		break
	case "JSON":
		logSource := getConfiguredLogSource()
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: logSource})))
	default:
		slog.Warn("Invalid log type", "provided", logTypeAsString, "default", DefaultLogType)
	}

	logLevel = getConfiguredLogLevel()
	slog.SetLogLoggerLevel(logLevel)
}
