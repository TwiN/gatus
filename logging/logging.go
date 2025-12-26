package logging

import (
	"errors"
	"log/slog"
	"os"
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
		slog.Info("Defaulting log level", "level", "INFO")
		return slog.LevelInfo
	} else if level, err := levelFromString(levelAsString); err != nil {
		slog.Warn("Invalid log level, using default", "provided", level, "default", "INFO")
		return slog.LevelInfo
	} else {
		if levelAsString == "FATAL" {
			slog.Warn("WARNING: FATAL log level has been deprecated and will be removed in v6.0.0")
			slog.Warn("WARNING: Please use the ERROR log level instead")
		}
		return level
	}
}

func getConfiguredLogSource() bool {
	logSourceAsString := os.Getenv(GatusLogSourceEnvVar)
	if len(logSourceAsString) == 0 {
		slog.Info("Defaulting log source to false")
		return false
	} else if logSourceAsString != "TRUE" && logSourceAsString != "FALSE" {
		slog.Warn("Invalid log source", "provided", logSourceAsString, "default", "FALSE")
		return false
	}
	return logSourceAsString == "TRUE"
}

func Configure() {
	logTypeAsString := os.Getenv(GatusConfigLogTypeEnvVar)
	switch logTypeAsString {
	case "", "TEXT":
		break
	case "JSON":
		logSource := getConfiguredLogSource()
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: logSource})))
	default:
		slog.Warn("Invalid log type", "provided", logTypeAsString, "default", "TEXT")
	}

	logLevel = getConfiguredLogLevel()
	slog.SetLogLoggerLevel(logLevel)
}
