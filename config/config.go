package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TwiN/deepmerge"
	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider"
	"github.com/TwiN/gatus/v5/config/connectivity"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/gatus/v5/config/ui"
	"github.com/TwiN/gatus/v5/config/web"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/security"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/util"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigurationFilePath is the default path that will be used to search for the configuration file
	// if a custom path isn't configured through the GATUS_CONFIG_PATH environment variable
	DefaultConfigurationFilePath = "config/config.yaml"

	// DefaultFallbackConfigurationFilePath is the default fallback path that will be used to search for the
	// configuration file if DefaultConfigurationFilePath didn't work
	DefaultFallbackConfigurationFilePath = "config/config.yml"
)

var (
	// ErrNoEndpointInConfig is an error returned when a configuration file or directory has no endpoints configured
	ErrNoEndpointInConfig = errors.New("configuration should contain at least 1 endpoint")

	// ErrConfigFileNotFound is an error returned when a configuration file could not be found
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrInvalidSecurityConfig is an error returned when the security configuration is invalid
	ErrInvalidSecurityConfig = errors.New("invalid security configuration")

	// errEarlyReturn is returned to break out of a loop from a callback early
	errEarlyReturn = errors.New("early escape")
)

// Config is the main configuration structure
type Config struct {
	// Debug Whether to enable debug logs
	Debug bool `yaml:"debug,omitempty"`

	// Metrics Whether to expose metrics at /metrics
	Metrics bool `yaml:"metrics,omitempty"`

	// SkipInvalidConfigUpdate Whether to make the application ignore invalid configuration
	// if the configuration file is updated while the application is running
	SkipInvalidConfigUpdate bool `yaml:"skip-invalid-config-update,omitempty"`

	// DisableMonitoringLock Whether to disable the monitoring lock
	// The monitoring lock is what prevents multiple endpoints from being processed at the same time.
	// Disabling this may lead to inaccurate response times
	DisableMonitoringLock bool `yaml:"disable-monitoring-lock,omitempty"`

	// Security Configuration for securing access to Gatus
	Security *security.Config `yaml:"security,omitempty"`

	// Alerting Configuration for alerting
	Alerting *alerting.Config `yaml:"alerting,omitempty"`

	// Endpoints List of endpoints to monitor
	Endpoints []*core.Endpoint `yaml:"endpoints,omitempty"`

	// Storage is the configuration for how the data is stored
	Storage *storage.Config `yaml:"storage,omitempty"`

	// Web is the web configuration for the application
	Web *web.Config `yaml:"web,omitempty"`

	// UI is the configuration for the UI
	UI *ui.Config `yaml:"ui,omitempty"`

	// Maintenance is the configuration for creating a maintenance window in which no alerts are sent
	Maintenance *maintenance.Config `yaml:"maintenance,omitempty"`

	// Remote is the configuration for remote Gatus instances
	// WARNING: This is in ALPHA and may change or be completely removed in the future
	Remote *remote.Config `yaml:"remote,omitempty"`

	// Connectivity is the configuration for connectivity
	Connectivity *connectivity.Config `yaml:"connectivity,omitempty"`

	configPath      string    // path to the file or directory from which config was loaded
	lastFileModTime time.Time // last modification time
}

func (config *Config) GetEndpointByKey(key string) *core.Endpoint {
	// TODO: Should probably add a mutex here to prevent concurrent access
	for i := 0; i < len(config.Endpoints); i++ {
		ep := config.Endpoints[i]
		if util.ConvertGroupAndEndpointNameToKey(ep.Group, ep.Name) == key {
			return ep
		}
	}
	return nil
}

// HasLoadedConfigurationBeenModified returns whether one of the file that the
// configuration has been loaded from has been modified since it was last read
func (config Config) HasLoadedConfigurationBeenModified() bool {
	lastMod := config.lastFileModTime.Unix()
	fileInfo, err := os.Stat(config.configPath)
	if err != nil {
		return false
	}
	if fileInfo.IsDir() {
		err = walkConfigDir(config.configPath, func(path string, d fs.DirEntry, err error) error {
			if info, err := d.Info(); err == nil && lastMod < info.ModTime().Unix() {
				return errEarlyReturn
			}
			return nil
		})
		return err == errEarlyReturn
	}
	return !fileInfo.ModTime().IsZero() && config.lastFileModTime.Unix() < fileInfo.ModTime().Unix()
}

// UpdateLastFileModTime refreshes Config.lastFileModTime
func (config *Config) UpdateLastFileModTime() {
	config.lastFileModTime = time.Now()
}

// LoadConfiguration loads the full configuration composed from the main configuration file
// and all composed configuration files
func LoadConfiguration(configPath string) (*Config, error) {
	var configBytes []byte
	var fileInfo os.FileInfo
	var usedConfigPath string
	// Figure out what config path we'll use (either configPath or the default config path)
	for _, configurationPath := range []string{configPath, DefaultConfigurationFilePath, DefaultFallbackConfigurationFilePath} {
		if len(configurationPath) == 0 {
			continue
		}
		var err error
		fileInfo, err = os.Stat(configurationPath)
		if err != nil {
			continue
		}
		usedConfigPath = configurationPath
		break
	}
	if len(usedConfigPath) == 0 {
		return nil, ErrConfigFileNotFound
	}
	var config *Config
	if fileInfo.IsDir() {
		err := walkConfigDir(configPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Printf("[config][LoadConfiguration] Error walking path=%s: %s", path, err)
				return err
			}
			log.Printf("[config][LoadConfiguration] Reading configuration from %s", path)
			data, err := os.ReadFile(path)
			if err != nil {
				log.Printf("[config][LoadConfiguration] Error reading configuration from %s: %s", path, err)
				return fmt.Errorf("error reading configuration from file %s: %w", path, err)
			}
			configBytes, err = deepmerge.YAML(configBytes, data)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("error reading configuration from directory %s: %w", usedConfigPath, err)
		}
	} else {
		log.Printf("[config][LoadConfiguration] Reading configuration from configFile=%s", configPath)
		if data, err := os.ReadFile(usedConfigPath); err != nil {
			return nil, err
		} else {
			configBytes = data
		}
	}
	if len(configBytes) == 0 {
		return nil, ErrConfigFileNotFound
	}
	config, err := parseAndValidateConfigBytes(configBytes)
	if err != nil {
		return nil, err
	}
	config.configPath = usedConfigPath
	config.UpdateLastFileModTime()
	return config, err
}

// walkConfigDir is a wrapper for filepath.WalkDir that strips directories and non-config files
func walkConfigDir(path string, fn fs.WalkDirFunc) error {
	if len(path) == 0 {
		// If the user didn't provide a directory, we'll just use the default config file, so we can return nil now.
		return nil
	}
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d == nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		return fn(path, d, err)
	})
}

// parseAndValidateConfigBytes parses a Gatus configuration file into a Config struct and validates its parameters
func parseAndValidateConfigBytes(yamlBytes []byte) (config *Config, err error) {
	// Replace $$ with __GATUS_LITERAL_DOLLAR_SIGN__ to prevent os.ExpandEnv from treating "$$" as if it was an
	// environment variable. This allows Gatus to support literal "$" in the configuration file.
	yamlBytes = []byte(strings.ReplaceAll(string(yamlBytes), "$$", "__GATUS_LITERAL_DOLLAR_SIGN__"))
	// Expand environment variables
	yamlBytes = []byte(os.ExpandEnv(string(yamlBytes)))
	// Replace __GATUS_LITERAL_DOLLAR_SIGN__ with "$" to restore the literal "$" in the configuration file
	yamlBytes = []byte(strings.ReplaceAll(string(yamlBytes), "__GATUS_LITERAL_DOLLAR_SIGN__", "$"))
	// Parse configuration file
	if err = yaml.Unmarshal(yamlBytes, &config); err != nil {
		return
	}
	// Check if the configuration file at least has endpoints configured
	if config == nil || config.Endpoints == nil || len(config.Endpoints) == 0 {
		err = ErrNoEndpointInConfig
	} else {
		validateAlertingConfig(config.Alerting, config.Endpoints, config.Debug)
		if err := validateSecurityConfig(config); err != nil {
			return nil, err
		}
		if err := validateEndpointsConfig(config); err != nil {
			return nil, err
		}
		if err := validateWebConfig(config); err != nil {
			return nil, err
		}
		if err := validateUIConfig(config); err != nil {
			return nil, err
		}
		if err := validateMaintenanceConfig(config); err != nil {
			return nil, err
		}
		if err := validateStorageConfig(config); err != nil {
			return nil, err
		}
		if err := validateRemoteConfig(config); err != nil {
			return nil, err
		}
		if err := validateConnectivityConfig(config); err != nil {
			return nil, err
		}
	}
	return
}

func validateConnectivityConfig(config *Config) error {
	if config.Connectivity != nil {
		return config.Connectivity.ValidateAndSetDefaults()
	}
	return nil
}

func validateRemoteConfig(config *Config) error {
	if config.Remote != nil {
		if err := config.Remote.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func validateStorageConfig(config *Config) error {
	if config.Storage == nil {
		config.Storage = &storage.Config{
			Type: storage.TypeMemory,
		}
	} else {
		if err := config.Storage.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func validateMaintenanceConfig(config *Config) error {
	if config.Maintenance == nil {
		config.Maintenance = maintenance.GetDefaultConfig()
	} else {
		if err := config.Maintenance.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func validateUIConfig(config *Config) error {
	if config.UI == nil {
		config.UI = ui.GetDefaultConfig()
	} else {
		if err := config.UI.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func validateWebConfig(config *Config) error {
	if config.Web == nil {
		config.Web = web.GetDefaultConfig()
	} else {
		return config.Web.ValidateAndSetDefaults()
	}
	return nil
}

func validateEndpointsConfig(config *Config) error {
	for _, endpoint := range config.Endpoints {
		if config.Debug {
			log.Printf("[config][validateEndpointsConfig] Validating endpoint '%s'", endpoint.Name)
		}
		if err := endpoint.ValidateAndSetDefaults(); err != nil {
			return fmt.Errorf("invalid endpoint %s: %w", endpoint.DisplayName(), err)
		}
	}
	log.Printf("[config][validateEndpointsConfig] Validated %d endpoints", len(config.Endpoints))
	return nil
}

func validateSecurityConfig(config *Config) error {
	if config.Security != nil {
		if config.Security.IsValid() {
			if config.Debug {
				log.Printf("[config][validateSecurityConfig] Basic security configuration has been validated")
			}
		} else {
			// If there was an attempt to configure security, then it must mean that some confidential or private
			// data are exposed. As a result, we'll force a panic because it's better to be safe than sorry.
			return ErrInvalidSecurityConfig
		}
	}
	return nil
}

// validateAlertingConfig validates the alerting configuration
// Note that the alerting configuration has to be validated before the endpoint configuration, because the default alert
// returned by provider.AlertProvider.GetDefaultAlert() must be parsed before core.Endpoint.ValidateAndSetDefaults()
// sets the default alert values when none are set.
func validateAlertingConfig(alertingConfig *alerting.Config, endpoints []*core.Endpoint, debug bool) {
	if alertingConfig == nil {
		log.Printf("[config][validateAlertingConfig] Alerting is not configured")
		return
	}
	alertTypes := []alert.Type{
		alert.TypeAWSSES,
		alert.TypeCustom,
		alert.TypeDiscord,
		alert.TypeGitHub,
		alert.TypeGitLab,
		alert.TypeGoogleChat,
		alert.TypeGotify,
		alert.TypeEmail,
		alert.TypeMatrix,
		alert.TypeMattermost,
		alert.TypeMessagebird,
		alert.TypeNtfy,
		alert.TypeOpsgenie,
		alert.TypePagerDuty,
		alert.TypePushover,
		alert.TypeSlack,
		alert.TypeTeams,
		alert.TypeTelegram,
		alert.TypeTwilio,
	}
	var validProviders, invalidProviders []alert.Type
	for _, alertType := range alertTypes {
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(alertType)
		if alertProvider != nil {
			if alertProvider.IsValid() {
				// Parse alerts with the provider's default alert
				if alertProvider.GetDefaultAlert() != nil {
					for _, endpoint := range endpoints {
						for alertIndex, endpointAlert := range endpoint.Alerts {
							if alertType == endpointAlert.Type {
								if debug {
									log.Printf("[config][validateAlertingConfig] Parsing alert %d with provider's default alert for provider=%s in endpoint=%s", alertIndex, alertType, endpoint.Name)
								}
								provider.ParseWithDefaultAlert(alertProvider.GetDefaultAlert(), endpointAlert)
							}
						}
					}
				}
				validProviders = append(validProviders, alertType)
			} else {
				log.Printf("[config][validateAlertingConfig] Ignoring provider=%s because configuration is invalid", alertType)
				invalidProviders = append(invalidProviders, alertType)
				alertingConfig.SetAlertingProviderToNil(alertProvider)
			}
		} else {
			invalidProviders = append(invalidProviders, alertType)
		}
	}
	log.Printf("[config][validateAlertingConfig] configuredProviders=%s; ignoredProviders=%s", validProviders, invalidProviders)
}
