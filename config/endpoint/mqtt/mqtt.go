package mqtt

import (
	"errors"
)

var (
	// ErrEndpointWithoutMQTTTopic is the error with which Gatus will panic if an endpoint with MQTT monitoring is configured without a topic.
	ErrEndpointWithoutMQTTTopic = errors.New("you must specify a topic for each MQTT endpoint")
)

type Config struct {
	Topic    string `yaml:"topic,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// Validate the SSH configuration
func (cfg *Config) Validate() error {
	if len(cfg.Topic) == 0 {
		return ErrEndpointWithoutMQTTTopic
	}
	return nil
}
