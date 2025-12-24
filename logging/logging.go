package logging

import (
	"os"

	"github.com/TwiN/logr"
)

const (
	GatusLogLevelEnvVar = "GATUS_LOG_LEVEL"
)

func Configure() {
	logLevelAsString := os.Getenv(GatusLogLevelEnvVar)
	if logLevel, err := logr.LevelFromString(logLevelAsString); err != nil {
		logr.SetThreshold(logr.LevelInfo)
		if len(logLevelAsString) == 0 {
			logr.Infof("[main.configureLogging] Defaulting log level to %s", logr.LevelInfo)
		} else {
			logr.Warnf("[main.configureLogging] Invalid log level '%s', defaulting to %s", logLevelAsString, logr.LevelInfo)
		}
	} else {
		logr.SetThreshold(logLevel)
		logr.Infof("[main.configureLogging] Log Level is set to %s", logr.GetThreshold())
	}
}
