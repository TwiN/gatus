package config

import (
	"github.com/TwinProduction/gatus/core"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

type Config struct {
	Services []*core.Service `yaml:"services"`
}

var config *Config

func Get() *Config {
	if config == nil {
		cfg, err := readConfigurationFile("config.yaml")
		if err != nil {
			panic(err)
		}
		config = cfg
	}
	return config
}

func readConfigurationFile(fileName string) (*Config, error) {
	if bytes, err := ioutil.ReadFile(fileName); err == nil {
		// file exists, so we'll parse it and return it
		return parseConfigBytes(bytes), nil
	} else {
		return nil, err
	}
	return &Config{}, nil
}

func parseConfigBytes(yamlBytes []byte) (config *Config) {
	yaml.Unmarshal(yamlBytes, &config)
	for _, service := range config.Services {
		if service.Interval == 0 {
			service.Interval = 10 * time.Second
		}
	}
	return config
}
