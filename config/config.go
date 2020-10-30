package config

import (
	"errors"
	"io/ioutil"
	"log"
	"os"

	"github.com/TwinProduction/gatus/alerting"
	"github.com/TwinProduction/gatus/alerting/provider"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/security"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultConfigurationFilePath is the default path that will be used to search for the configuration file
	// if a custom path isn't configured through the GATUS_CONFIG_FILE environment variable
	DefaultConfigurationFilePath = "config/config.yaml"

	// DefaultFallbackConfigurationFilePath is the default fallback path that will be used to search for the
	// configuration file if DefaultConfigurationFilePath didn't work
	DefaultFallbackConfigurationFilePath = "config/config.yml"
)

var (
	// ErrNoServiceInConfig is an error returned when a configuration file has no services configured
	ErrNoServiceInConfig = errors.New("configuration file should contain at least 1 service")

	// ErrConfigFileNotFound is an error returned when the configuration file could not be found
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrConfigNotLoaded is an error returned when an attempt to Get() the configuration before loading it is made
	ErrConfigNotLoaded = errors.New("configuration is nil")

	// ErrInvalidSecurityConfig is an error returned when the security configuration is invalid
	ErrInvalidSecurityConfig = errors.New("invalid security configuration")

	config *Config
)

// Config is the main configuration structure
type Config struct {
	// Debug Whether to enable debug logs
	Debug bool `yaml:"debug"`

	// Metrics Whether to expose metrics at /metrics
	Metrics bool `yaml:"metrics"`

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

	//AutoDiscoverK8SServices to discover services to monitor
	AutoDiscoverK8SServices bool `yaml:"auto-discover-k8s-services"`

	//K8SServiceSuffix to append to service name
	K8SServiceSuffix string `yaml:"k8s-service-suffix"`

	K8SServiceConfig core.Service `yaml:"k8s-service-config"`

	ExcludeSuffix []string `yaml:"exclude-suffix"`

	K8sClusterMode string `yaml:"k8s-cluster-mode"`
}

// Get returns the configuration, or panics if the configuration hasn't loaded yet
func Get() *Config {
	if config == nil {
		panic(ErrConfigNotLoaded)
	}
	return config
}

// Load loads a custom configuration file
func Load(configFile string) error {
	log.Printf("[config][Load] Reading configuration from configFile=%s", configFile)
	cfg, err := readConfigurationFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrConfigFileNotFound
		}
		return err
	}
	config = cfg
	return nil
}

// LoadDefaultConfiguration loads the default configuration file
func LoadDefaultConfiguration() error {
	err := Load(DefaultConfigurationFilePath)
	if err != nil {
		if err == ErrConfigFileNotFound {
			return Load(DefaultFallbackConfigurationFilePath)
		}
		return err
	}
	return nil
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
	// Check if the configuration file at least has services.
	if config == nil || config.Services == nil || len(config.Services) == 0 {
		err = ErrNoServiceInConfig
	} else {
		validateAlertingConfig(config)
		validateSecurityConfig(config)
		validateServicesConfig(config)
	}
	return
}

func validateServicesConfig(config *Config) {
	for _, service := range config.Services {
		if config.Debug {
			log.Printf("[config][validateServicesConfig] Validating service '%s'", service.Name)
		}
		service.ValidateAndSetDefaults()
	}
	log.Printf("[config][validateServicesConfig] Validated %d services", len(config.Services))
}

func validateSecurityConfig(config *Config) {
	if config.Security != nil {
		if config.Security.IsValid() {
			if config.Debug {
				log.Printf("[config][validateSecurityConfig] Basic security configuration has been validated")
			}
		} else {
			// If there was an attempt to configure security, then it must mean that some confidential or private
			// data are exposed. As a result, we'll force a panic because it's better to be safe than sorry.
			panic(ErrInvalidSecurityConfig)
		}
	}
}

func validateAlertingConfig(config *Config) {
	if config.Alerting == nil {
		log.Printf("[config][validateAlertingConfig] Alerting is not configured")
		return
	}
	alertTypes := []core.AlertType{
		core.SlackAlert,
		core.TwilioAlert,
		core.PagerDutyAlert,
		core.CustomAlert,
	}
	var validProviders, invalidProviders []core.AlertType
	for _, alertType := range alertTypes {
		alertProvider := GetAlertingProviderByAlertType(config, alertType)
		if alertProvider != nil {
			if alertProvider.IsValid() {
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

// GetAlertingProviderByAlertType returns an provider.AlertProvider by its corresponding core.AlertType
func GetAlertingProviderByAlertType(config *Config, alertType core.AlertType) provider.AlertProvider {
	switch alertType {
	case core.SlackAlert:
		if config.Alerting.Slack == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Alerting.Slack
	case core.TwilioAlert:
		if config.Alerting.Twilio == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Alerting.Twilio
	case core.PagerDutyAlert:
		if config.Alerting.PagerDuty == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Alerting.PagerDuty
	case core.CustomAlert:
		if config.Alerting.Custom == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Alerting.Custom
	}
	return nil
}
