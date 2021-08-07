package config

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/TwinProduction/gatus/alerting"
	"github.com/TwinProduction/gatus/alerting/alert"
	"github.com/TwinProduction/gatus/alerting/provider"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/k8s"
	"github.com/TwinProduction/gatus/security"
	"github.com/TwinProduction/gatus/storage"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultConfigurationFilePath is the default path that will be used to search for the configuration file
	// if a custom path isn't configured through the GATUS_CONFIG_FILE environment variable
	DefaultConfigurationFilePath = "config/config.yaml"

	// DefaultFallbackConfigurationFilePath is the default fallback path that will be used to search for the
	// configuration file if DefaultConfigurationFilePath didn't work
	DefaultFallbackConfigurationFilePath = "config/config.yml"

	// DefaultAddress is the default address the service will bind to
	DefaultAddress = "0.0.0.0"

	// DefaultPort is the default port the service will listen on
	DefaultPort = 8080
)

var (
	// ErrNoServiceInConfig is an error returned when a configuration file has no services configured
	ErrNoServiceInConfig = errors.New("configuration file should contain at least 1 service")

	// ErrConfigFileNotFound is an error returned when the configuration file could not be found
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrInvalidSecurityConfig is an error returned when the security configuration is invalid
	ErrInvalidSecurityConfig = errors.New("invalid security configuration")
)

// Config is the main configuration structure
type Config struct {
	// Debug Whether to enable debug logs
	Debug bool `yaml:"debug"`

	// Metrics Whether to expose metrics at /metrics
	Metrics bool `yaml:"metrics"`

	// SkipInvalidConfigUpdate Whether to make the application ignore invalid configuration
	// if the configuration file is updated while the application is running
	SkipInvalidConfigUpdate bool `yaml:"skip-invalid-config-update"`

	// DisableMonitoringLock Whether to disable the monitoring lock
	// The monitoring lock is what prevents multiple services from being processed at the same time.
	// Disabling this may lead to inaccurate response times
	DisableMonitoringLock bool `yaml:"disable-monitoring-lock"`

	// Security Configuration for securing access to Gatus
	Security *security.Config `yaml:"security"`

	// Alerting Configuration for alerting
	Alerting *alerting.Config `yaml:"alerting"`

	// Services List of services to monitor
	Services []*core.Service `yaml:"services"`

	// Kubernetes is the Kubernetes configuration
	Kubernetes *k8s.Config `yaml:"kubernetes"`

	// Storage is the configuration for how the data is stored
	Storage *storage.Config `yaml:"storage"`

	// Web is the configuration for the web listener
	Web *WebConfig `yaml:"web"`

	filePath        string    // path to the file from which config was loaded from
	lastFileModTime time.Time // last modification time
}

// HasLoadedConfigurationFileBeenModified returns whether the file that the
// configuration has been loaded from has been modified since it was last read
func (config Config) HasLoadedConfigurationFileBeenModified() bool {
	if fileInfo, err := os.Stat(config.filePath); err == nil {
		if !fileInfo.ModTime().IsZero() {
			return config.lastFileModTime.Unix() != fileInfo.ModTime().Unix()
		}
	}
	return false
}

// UpdateLastFileModTime refreshes Config.lastFileModTime
func (config *Config) UpdateLastFileModTime() {
	if fileInfo, err := os.Stat(config.filePath); err == nil {
		if !fileInfo.ModTime().IsZero() {
			config.lastFileModTime = fileInfo.ModTime()
		}
	}
}

// Load loads a custom configuration file
// Note that the misconfiguration of some fields may lead to panics. This is on purpose.
func Load(configFile string) (*Config, error) {
	log.Printf("[config][Load] Reading configuration from configFile=%s", configFile)
	cfg, err := readConfigurationFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigFileNotFound
		}
		return nil, err
	}
	cfg.filePath = configFile
	cfg.UpdateLastFileModTime()
	return cfg, nil
}

// LoadDefaultConfiguration loads the default configuration file
func LoadDefaultConfiguration() (*Config, error) {
	cfg, err := Load(DefaultConfigurationFilePath)
	if err != nil {
		if err == ErrConfigFileNotFound {
			return Load(DefaultFallbackConfigurationFilePath)
		}
		return nil, err
	}
	return cfg, nil
}

func readConfigurationFile(fileName string) (config *Config, err error) {
	var bytes []byte
	if bytes, err = ioutil.ReadFile(fileName); err == nil {
		// file exists, so we'll parse it and return it
		return parseAndValidateConfigBytes(bytes)
	}
	return
}

func parseAndValidateConfigBytes(yamlBytes []byte) (config *Config, err error) {
	// Expand environment variables
	yamlBytes = []byte(os.ExpandEnv(string(yamlBytes)))
	// Parse configuration file
	err = yaml.Unmarshal(yamlBytes, &config)
	if err != nil {
		return
	}
	// Check if the configuration file at least has services configured or Kubernetes auto discovery enabled
	if config == nil || ((config.Services == nil || len(config.Services) == 0) && (config.Kubernetes == nil || !config.Kubernetes.AutoDiscover)) {
		err = ErrNoServiceInConfig
	} else {
		// Note that the functions below may panic, and this is on purpose to prevent Gatus from starting with
		// invalid configurations
		validateAlertingConfig(config.Alerting, config.Services, config.Debug)
		if err := validateSecurityConfig(config); err != nil {
			return nil, err
		}
		if err := validateServicesConfig(config); err != nil {
			return nil, err
		}
		if err := validateKubernetesConfig(config); err != nil {
			return nil, err
		}
		if err := validateWebConfig(config); err != nil {
			return nil, err
		}
		if err := validateStorageConfig(config); err != nil {
			return nil, err
		}
	}
	return
}

func validateStorageConfig(config *Config) error {
	if config.Storage == nil {
		config.Storage = &storage.Config{
			Type: storage.TypeMemory,
		}
	}
	err := storage.Initialize(config.Storage)
	if err != nil {
		return err
	}
	// Remove all ServiceStatus that represent services which no longer exist in the configuration
	var keys []string
	for _, service := range config.Services {
		keys = append(keys, service.Key())
	}
	numberOfServiceStatusesDeleted := storage.Get().DeleteAllServiceStatusesNotInKeys(keys)
	if numberOfServiceStatusesDeleted > 0 {
		log.Printf("[config][validateStorageConfig] Deleted %d service statuses because their matching services no longer existed", numberOfServiceStatusesDeleted)
	}
	return nil
}

func validateWebConfig(config *Config) error {
	if config.Web == nil {
		config.Web = &WebConfig{Address: DefaultAddress, Port: DefaultPort}
	} else {
		return config.Web.validateAndSetDefaults()
	}
	return nil
}

// deprecated
// I don't like the current implementation.
func validateKubernetesConfig(config *Config) error {
	if config.Kubernetes != nil && config.Kubernetes.AutoDiscover {
		log.Println("WARNING - The Kubernetes integration is planned to be removed in v3.0.0. If you're seeing this message, it's because you're currently using it, and you may want to give your opinion at https://github.com/TwinProduction/gatus/discussions/135")
		if config.Kubernetes.ServiceTemplate == nil {
			return errors.New("kubernetes.service-template cannot be nil")
		}
		if config.Debug {
			log.Println("[config][validateKubernetesConfig] Automatically discovering Kubernetes services...")
		}
		discoveredServices, err := k8s.DiscoverServices(config.Kubernetes)
		if err != nil {
			return err
		}
		config.Services = append(config.Services, discoveredServices...)
		log.Printf("[config][validateKubernetesConfig] Discovered %d Kubernetes services", len(discoveredServices))
	}
	return nil
}

func validateServicesConfig(config *Config) error {
	for _, service := range config.Services {
		if config.Debug {
			log.Printf("[config][validateServicesConfig] Validating service '%s'", service.Name)
		}
		if err := service.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	log.Printf("[config][validateServicesConfig] Validated %d services", len(config.Services))
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
// Note that the alerting configuration has to be validated before the service configuration, because the default alert
// returned by provider.AlertProvider.GetDefaultAlert() must be parsed before core.Service.ValidateAndSetDefaults()
// sets the default alert values when none are set.
func validateAlertingConfig(alertingConfig *alerting.Config, services []*core.Service, debug bool) {
	if alertingConfig == nil {
		log.Printf("[config][validateAlertingConfig] Alerting is not configured")
		return
	}
	alertTypes := []alert.Type{
		alert.TypeCustom,
		alert.TypeDiscord,
		alert.TypeMattermost,
		alert.TypeMessagebird,
		alert.TypePagerDuty,
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
					for _, service := range services {
						for alertIndex, serviceAlert := range service.Alerts {
							if alertType == serviceAlert.Type {
								if debug {
									log.Printf("[config][validateAlertingConfig] Parsing alert %d with provider's default alert for provider=%s in service=%s", alertIndex, alertType, service.Name)
								}
								provider.ParseWithDefaultAlert(alertProvider.GetDefaultAlert(), serviceAlert)
							}
						}
					}
				}
				validProviders = append(validProviders, alertType)
			} else {
				log.Printf("[config][validateAlertingConfig] Ignoring provider=%s because configuration is invalid", alertType)
				invalidProviders = append(invalidProviders, alertType)
			}
		} else {
			invalidProviders = append(invalidProviders, alertType)
		}
	}
	log.Printf("[config][validateAlertingConfig] configuredProviders=%s; ignoredProviders=%s", validProviders, invalidProviders)
}
