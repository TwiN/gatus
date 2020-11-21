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
// in given in DefaultWebContext, DefaultAddress and DefaultPort if necessary
func (web *webConfig) validateAndSetDefaults() {
	if len(web.Address) == 0 {
		web.Address = DefaultAddress
	}
	if web.Port == 0 {
		web.Port = DefaultPort
	} else if web.Port < 0 || web.Port > math.MaxUint16 {
		panic(fmt.Sprintf("port has an invalid: value should be between %d and %d", 0, math.MaxUint16))
	}

	web.ContextRoot = validateAndBuild(web.ContextRoot)
}

// validateAndBuild validates and builds a checked
// path for the context root
func validateAndBuild(contextRoot string) string {
	trimedContextRoot := strings.Trim(contextRoot, "/")

	if len(trimedContextRoot) == 0 {
		return DefaultContextRoot
	} else {
		url, err := url.Parse(trimedContextRoot)
		if err != nil {
			panic(fmt.Sprintf("Invalid context root %s - error: %s.", contextRoot, err))
		}
		if url.Path != trimedContextRoot {
			panic(fmt.Sprintf("Invalid context root %s, simple path required.", contextRoot))
		}

		return "/" + strings.Trim(url.Path, "/") + "/"
	}
}

// SocketAddress returns the combination of the Address and the Port
func (web *webConfig) SocketAddress() string {
	return fmt.Sprintf("%s:%d", web.Address, web.Port)
}

// PrependWithContextRoot appends the given string to the context root
// PrependWithContextRoot takes care of having only one "/" character at
// the join point and exactly on "/" at the end
func (web *webConfig) PrependWithContextRoot(fragment string) string {
	return web.ContextRoot + strings.Trim(fragment, "/") + "/"
}
