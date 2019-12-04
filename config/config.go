package config

import (
	"errors"
	"github.com/TwinProduction/gatus/core"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type Config struct {
	Metrics  bool            `yaml:"metrics"`
	Services []*core.Service `yaml:"services"`
}

var (
	ErrNoServiceInConfig  = errors.New("configuration file should contain at least 1 service")
	ErrConfigFileNotFound = errors.New("configuration file not found")
	ErrConfigNotLoaded    = errors.New("configuration is nil")
	config                *Config
)

func Get() *Config {
	if config == nil {
		panic(ErrConfigNotLoaded)
	}
	return config
}

func Load(configFile string) error {
	log.Printf("[config][Load] Attempting to load config from configFile=%s", configFile)
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
	err := Load("config.yaml")
	if err != nil {
		if err == ErrConfigFileNotFound {
			return Load("config.yml")
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
	err = yaml.Unmarshal(yamlBytes, &config)
	// Check if the configuration file at least has services.
	if config == nil || len(config.Services) == 0 {
		err = ErrNoServiceInConfig
	} else {
		// Set the default values if they aren't set
		for _, service := range config.Services {
			if service.Interval == 0 {
				service.Interval = 10 * time.Second
			}
		}
	}
	return
}
