package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/controller"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
	"github.com/TwiN/logr"
)

const (
	GatusConfigPathEnvVar = "GATUS_CONFIG_PATH"
	GatusConfigFileEnvVar = "GATUS_CONFIG_FILE" // Deprecated in favor of GatusConfigPathEnvVar
	GatusLogLevelEnvVar   = "GATUS_LOG_LEVEL"
)

func main() {
	if delayInSeconds, _ := strconv.Atoi(os.Getenv("GATUS_DELAY_START_SECONDS")); delayInSeconds > 0 {
		logr.Infof("Delaying start by %d seconds", delayInSeconds)
		time.Sleep(time.Duration(delayInSeconds) * time.Second)
	}
	configureLogging()
	cfg, err := loadConfiguration()
	if err != nil {
		panic(err)
	}
	initializeStorage(cfg)
	start(cfg)
	// Wait for termination signal
	signalChannel := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		logr.Info("Received termination signal, attempting to gracefully shut down")
		stop(cfg)
		save()
		done <- true
	}()
	<-done
	logr.Info("Shutting down")
}

func start(cfg *config.Config) {
	go controller.Handle(cfg)
	metrics.InitializePrometheusMetrics(cfg, nil)
	watchdog.Monitor(cfg)
	go listenToConfigurationFileChanges(cfg)
}

func stop(cfg *config.Config) {
	watchdog.Shutdown(cfg)
	controller.Shutdown()
	metrics.UnregisterPrometheusMetrics()
	closeTunnels(cfg)
}

func save() {
	if err := store.Get().Save(); err != nil {
		logr.Errorf("Failed to save storage provider: %s", err.Error())
	}
}

func configureLogging() {
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

func loadConfiguration() (*config.Config, error) {
	configPath := os.Getenv(GatusConfigPathEnvVar)
	// Backwards compatibility
	if len(configPath) == 0 {
		if configPath = os.Getenv(GatusConfigFileEnvVar); len(configPath) > 0 {
			logr.Warnf("WARNING: %s is deprecated. Please use %s instead.", GatusConfigFileEnvVar, GatusConfigPathEnvVar)
		}
	}
	return config.LoadConfiguration(configPath)
}

// initializeStorage initializes the storage provider
//
// Q: "TwiN, why are you putting this here? Wouldn't it make more sense to have this in the config?!"
// A: Yes. Yes it would make more sense to have it in the config package. But I don't want to import
// the massive SQL dependencies just because I want to import the config, so here we are.
func initializeStorage(cfg *config.Config) {
	err := store.Initialize(cfg.Storage)
	if err != nil {
		panic(err)
	}
	// Remove all SuiteStatuses that represent suites which no longer exist in the configuration
	var suiteKeys []string
	for _, suite := range cfg.Suites {
		suiteKeys = append(suiteKeys, suite.Key())
	}
	numberOfSuiteStatusesDeleted := store.Get().DeleteAllSuiteStatusesNotInKeys(suiteKeys)
	if numberOfSuiteStatusesDeleted > 0 {
		logr.Infof("[main.initializeStorage] Deleted %d suite statuses because their matching suites no longer existed", numberOfSuiteStatusesDeleted)
	}
	// Remove all EndpointStatus that represent endpoints which no longer exist in the configuration
	var keys []string
	for _, ep := range cfg.Endpoints {
		keys = append(keys, ep.Key())
	}
	for _, ee := range cfg.ExternalEndpoints {
		keys = append(keys, ee.Key())
	}
	// Also add endpoints that are part of suites
	for _, suite := range cfg.Suites {
		for _, ep := range suite.Endpoints {
			keys = append(keys, ep.Key())
		}
	}
	logr.Infof("[main.initializeStorage] Total endpoint keys to preserve: %d", len(keys))
	numberOfEndpointStatusesDeleted := store.Get().DeleteAllEndpointStatusesNotInKeys(keys)
	if numberOfEndpointStatusesDeleted > 0 {
		logr.Infof("[main.initializeStorage] Deleted %d endpoint statuses because their matching endpoints no longer existed", numberOfEndpointStatusesDeleted)
	}
	// Clean up the triggered alerts from the storage provider and load valid triggered endpoint alerts
	numberOfPersistedTriggeredAlertsLoaded := 0
	for _, ep := range cfg.Endpoints {
		var checksums []string
		for _, alert := range ep.Alerts {
			if alert.IsEnabled() {
				checksums = append(checksums, alert.Checksum())
			}
		}
		numberOfTriggeredAlertsDeleted := store.Get().DeleteAllTriggeredAlertsNotInChecksumsByEndpoint(ep, checksums)
		if numberOfTriggeredAlertsDeleted > 0 {
			logr.Debugf("[main.initializeStorage] Deleted %d triggered alerts for endpoint with key=%s because their configurations have been changed or deleted", numberOfTriggeredAlertsDeleted, ep.Key())
		}
		for _, alert := range ep.Alerts {
			exists, resolveKey, numberOfSuccessesInARow, err := store.Get().GetTriggeredEndpointAlert(ep, alert)
			if err != nil {
				logr.Errorf("[main.initializeStorage] Failed to get triggered alert for endpoint with key=%s: %s", ep.Key(), err.Error())
				continue
			}
			if exists {
				alert.Triggered, alert.ResolveKey = true, resolveKey
				ep.NumberOfSuccessesInARow, ep.NumberOfFailuresInARow = numberOfSuccessesInARow, alert.FailureThreshold
				numberOfPersistedTriggeredAlertsLoaded++
			}
		}
	}
	for _, ee := range cfg.ExternalEndpoints {
		var checksums []string
		for _, alert := range ee.Alerts {
			if alert.IsEnabled() {
				checksums = append(checksums, alert.Checksum())
			}
		}
		convertedEndpoint := ee.ToEndpoint()
		numberOfTriggeredAlertsDeleted := store.Get().DeleteAllTriggeredAlertsNotInChecksumsByEndpoint(convertedEndpoint, checksums)
		if numberOfTriggeredAlertsDeleted > 0 {
			logr.Debugf("[main.initializeStorage] Deleted %d triggered alerts for endpoint with key=%s because their configurations have been changed or deleted", numberOfTriggeredAlertsDeleted, ee.Key())
		}
		for _, alert := range ee.Alerts {
			exists, resolveKey, numberOfSuccessesInARow, err := store.Get().GetTriggeredEndpointAlert(convertedEndpoint, alert)
			if err != nil {
				logr.Errorf("[main.initializeStorage] Failed to get triggered alert for endpoint with key=%s: %s", ee.Key(), err.Error())
				continue
			}
			if exists {
				alert.Triggered, alert.ResolveKey = true, resolveKey
				ee.NumberOfSuccessesInARow, ee.NumberOfFailuresInARow = numberOfSuccessesInARow, alert.FailureThreshold
				numberOfPersistedTriggeredAlertsLoaded++
			}
		}
	}
	// Load persisted triggered alerts for suite endpoints
	for _, suite := range cfg.Suites {
		for _, ep := range suite.Endpoints {
			var checksums []string
			for _, alert := range ep.Alerts {
				if alert.IsEnabled() {
					checksums = append(checksums, alert.Checksum())
				}
			}
			numberOfTriggeredAlertsDeleted := store.Get().DeleteAllTriggeredAlertsNotInChecksumsByEndpoint(ep, checksums)
			if numberOfTriggeredAlertsDeleted > 0 {
				logr.Debugf("[main.initializeStorage] Deleted %d triggered alerts for suite endpoint with key=%s because their configurations have been changed or deleted", numberOfTriggeredAlertsDeleted, ep.Key())
			}
			for _, alert := range ep.Alerts {
				exists, resolveKey, numberOfSuccessesInARow, err := store.Get().GetTriggeredEndpointAlert(ep, alert)
				if err != nil {
					logr.Errorf("[main.initializeStorage] Failed to get triggered alert for suite endpoint with key=%s: %s", ep.Key(), err.Error())
					continue
				}
				if exists {
					alert.Triggered, alert.ResolveKey = true, resolveKey
					ep.NumberOfSuccessesInARow, ep.NumberOfFailuresInARow = numberOfSuccessesInARow, alert.FailureThreshold
					numberOfPersistedTriggeredAlertsLoaded++
				}
			}
		}
	}
	if numberOfPersistedTriggeredAlertsLoaded > 0 {
		logr.Infof("[main.initializeStorage] Loaded %d persisted triggered alerts", numberOfPersistedTriggeredAlertsLoaded)
	}
}

func closeTunnels(cfg *config.Config) {
	if cfg.Tunneling != nil {
		if err := cfg.Tunneling.Close(); err != nil {
			logr.Errorf("[main.closeTunnels] Error closing SSH tunnels: %v", err)
		}
	}
}

func listenToConfigurationFileChanges(cfg *config.Config) {
	for {
		time.Sleep(30 * time.Second)
		if cfg.HasLoadedConfigurationBeenModified() {
			logr.Info("[main.listenToConfigurationFileChanges] Configuration file has been modified")
			stop(cfg)
			time.Sleep(time.Second) // Wait a bit to make sure everything is done.
			save()
			updatedConfig, err := loadConfiguration()
			if err != nil {
				if cfg.SkipInvalidConfigUpdate {
					logr.Errorf("[main.listenToConfigurationFileChanges] Failed to load new configuration: %s", err.Error())
					logr.Error("[main.listenToConfigurationFileChanges] The configuration file was updated, but it is not valid. The old configuration will continue being used.")
					// Update the last file modification time to avoid trying to process the same invalid configuration again
					cfg.UpdateLastFileModTime()
					continue
				} else {
					panic(err)
				}
			}
			store.Get().Close()
			initializeStorage(updatedConfig)
			start(updatedConfig)
			return
		}
	}
}
