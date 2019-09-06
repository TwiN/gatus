package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Services []Service `yaml:"services"`
}

type Service struct {
	Name             string      `yaml:"name"`
	Url              string      `yaml:"url"`
	Interval         uint        `yaml:"interval"`
	FailureThreshold uint        `yaml:"failure-threshold"`
	Conditions       []Condition `yaml:"conditions"`
}

type Condition string

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
	return config
}
