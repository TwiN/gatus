package config

import (
	"fmt"
	"math"
	"net/url"
	"strings"
)

// webConfig is the structure which supports the configuration of the endpoint
// which provides access to the web frontend
type webConfig struct {
	// Address to listen on (defaults to 0.0.0.0 specified by DefaultAddress)
	Address string `yaml:"address"`

	// Port to listen on (default to 8080 specified by DefaultPort)
	Port int `yaml:"port"`

	// ContextRoot set the root context for the web application
	ContextRoot string `yaml:"context-root"`
}

// validateAndSetDefaults checks and sets missing values based on the defaults
// in given in DefaultAddress and DefaultPort if necessary
func (web *webConfig) validateAndSetDefaults() {
	if len(web.Address) == 0 {
		web.Address = DefaultAddress
	}
	if web.Port == 0 {
		web.Port = DefaultPort
	} else if web.Port < 0 || web.Port > math.MaxUint16 {
		panic(fmt.Sprintf("port has an invalid: value should be between %d and %d", 0, math.MaxUint16))
	}
	if len(web.ContextRoot) == 0 {
		web.ContextRoot = DefaultContextRoot
	} else {
		url, err := url.Parse(web.ContextRoot)
		if err != nil {
			panic(fmt.Sprintf("Invalid context root %s - error: %s.", web.ContextRoot, err))
		}
		if url.Path != web.ContextRoot {
			panic(fmt.Sprintf("Invalid context root %s, simple path required.", web.ContextRoot))
		}
		web.ContextRoot = strings.TrimRight(url.Path, "/") + "/"
	}
}

// SocketAddress returns the combination of the Address and the Port
func (web *webConfig) SocketAddress() string {
	return fmt.Sprintf("%s:%d", web.Address, web.Port)
}

// AppendToContexRoot appends the given string to the context root
// AppendToContexRoot takes care of having only one "/" character at
// the join point and exactly on "/" at the end
func (web *webConfig) AppendToContexRoot(fragment string) string {
	return web.ContextRoot + strings.Trim(fragment, "/") + "/"
}
