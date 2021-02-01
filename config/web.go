package config

import (
	"fmt"
	"math"
)

// WebConfig is the structure which supports the configuration of the endpoint
// which provides access to the web frontend
type WebConfig struct {
	// Address to listen on (defaults to 0.0.0.0 specified by DefaultAddress)
	Address string `yaml:"address"`

	// Port to listen on (default to 8080 specified by DefaultPort)
	Port int `yaml:"port"`
}

// validateAndSetDefaults checks and sets the default values for fields that are not set
func (web *WebConfig) validateAndSetDefaults() {
	// Validate the Address
	if len(web.Address) == 0 {
		web.Address = DefaultAddress
	}
	// Validate the Port
	if web.Port == 0 {
		web.Port = DefaultPort
	} else if web.Port < 0 || web.Port > math.MaxUint16 {
		panic(fmt.Sprintf("invalid port: value should be between %d and %d", 0, math.MaxUint16))
	}
}

// SocketAddress returns the combination of the Address and the Port
func (web *WebConfig) SocketAddress() string {
	return fmt.Sprintf("%s:%d", web.Address, web.Port)
}
