package config

import (
	"errors"
	"github.com/TwinProduction/gatus/alerting"
	"github.com/TwinProduction/gatus/alerting/provider"
	"github.com/TwinProduction/gatus/core"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

const (
	DefaultConfigurationFilePath = "config/config.yaml"
)

var (
	ErrNoServiceInConfig  = errors.New("configuration file should contain at least 1 service")
	ErrConfigFileNotFound = errors.New("configuration file not found")
	ErrConfigNotLoaded    = errors.New("configuration is nil")
	config                *Config
)

type Config struct {
	Metrics  bool             `yaml:"metrics"`
	Debug    bool             `yaml:"debug"`
	Alerting *alerting.Config `yaml:"alerting"`
	Services []*core.Service  `yaml:"services"`
}

func Get() *Config {
	if config == nil {
		panic(ErrConfigNotLoaded)
	}
	return config
}

func Load(configFile string) error {
	log.Printf("[config][Load] Reading configuration from configFile=%s", configFile)
	cfg, err := readConfigurationFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrConfigFileNotFound
		} else {
			return err
		}
	}
	config = cfg
	return nil
}

func LoadDefaultConfiguration() error {
	err := Load(DefaultConfigurationFilePath)
	if err != nil {
		if err == ErrConfigFileNotFound {
			return Load("config/config.yml")
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
	// Check if the configuration file at least has services.
	if config == nil || config.Services == nil || len(config.Services) == 0 {
		err = ErrNoServiceInConfig
	} else {
		validateAlertingConfig(config)
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
	//if config.Alerting.Slack != nil {
	//	if config.Alerting.Slack.IsValid() {
	//		validProviders = append(validProviders, core.SlackAlert)
	//	} else {
	//		log.Printf("[config][validateAlertingConfig] Ignoring provider=%s because configuration is invalid", core.SlackAlert)
	//		invalidProviders = append(invalidProviders, core.SlackAlert)
	//	}
	//} else {
	//	invalidProviders = append(invalidProviders, core.SlackAlert)
	//}
	//if config.Alerting.Twilio != nil {
	//	if config.Alerting.Twilio.IsValid() {
	//		validProviders = append(validProviders, core.TwilioAlert)
	//	} else {
	//		log.Printf("[config][validateAlertingConfig] Ignoring provider=%s because configuration is invalid", core.TwilioAlert)
	//		invalidProviders = append(invalidProviders, core.TwilioAlert)
	//	}
	//} else {
	//	invalidProviders = append(invalidProviders, core.TwilioAlert)
	//}
	//if config.Alerting.PagerDuty != nil {
	//	if config.Alerting.PagerDuty.IsValid() {
	//		validProviders = append(validProviders, core.PagerDutyAlert)
	//	} else {
	//		log.Printf("[config][validateAlertingConfig] Ignoring provider=%s because configuration is invalid", core.PagerDutyAlert)
	//		invalidProviders = append(invalidProviders, core.PagerDutyAlert)
	//	}
	//} else {
	//	invalidProviders = append(invalidProviders, core.PagerDutyAlert)
	//}
	//if config.Alerting.Custom != nil {
	//	if config.Alerting.Custom.IsValid() {
	//		validProviders = append(validProviders, core.CustomAlert)
	//	} else {
	//		log.Printf("[config][validateAlertingConfig] Ignoring provider=%s because configuration is invalid", core.CustomAlert)
	//		invalidProviders = append(invalidProviders, core.CustomAlert)
	//	}
	//} else {
	//	invalidProviders = append(invalidProviders, core.CustomAlert)
	//}
	log.Printf("[config][validateAlertingConfig] configuredProviders=%s; ignoredProviders=%s", validProviders, invalidProviders)
}

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
