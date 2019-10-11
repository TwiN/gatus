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
		ReadConfigurationFile("config.yaml")
	}
	return config
}

func ReadConfigurationFile(fileName string) *Config {
	config = &Config{}
	if bytes, err := ioutil.ReadFile(fileName); err == nil { // file exists
		return ParseConfigBytes(bytes)
	} else {
		panic(err)
	}
	return config
}

func ParseConfigBytes(yamlBytes []byte) *Config {
	config = &Config{}
	yaml.Unmarshal(yamlBytes, config)
	for _, service := range config.Services {
		if service.Interval == 0 {
			service.Interval = 10 * time.Second
		}
	}
	return config
}
