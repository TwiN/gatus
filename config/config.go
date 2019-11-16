package config

import (
	"errors"
	"github.com/TwinProduction/gatus/core"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"time"
)

type Config struct {
	Metrics  bool            `yaml:"metrics"`
	Services []*core.Service `yaml:"services"`
}

var (
	ErrNoServiceInConfig = errors.New("configuration file should contain at least 1 service")
	config               *Config
)

func Get() *Config {
	if config == nil {
		cfg, err := readConfigurationFile("config.yaml")
		if err != nil {
			if os.IsNotExist(err) {
				cfg, err = readConfigurationFile("config.yml")
				if err != nil {
					panic(err)
				}
			} else {
				panic(err)
			}
		}
		config = cfg
	}
	return config
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
