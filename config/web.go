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

// validateAndSetDefaults checks and sets the default values for fields that are not set
func (web *webConfig) validateAndSetDefaults() {
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
	// Validate the ContextRoot
	if len(web.ContextRoot) == 0 {
		web.ContextRoot = DefaultContextRoot
	} else {
		trimmedContextRoot := strings.Trim(web.ContextRoot, "/")
		if len(trimmedContextRoot) == 0 {
			web.ContextRoot = DefaultContextRoot
			return
		}
		rootContextURL, err := url.Parse(trimmedContextRoot)
		if err != nil {
			panic("invalid context root:" + err.Error())
		}
		if rootContextURL.Path != trimmedContextRoot {
			panic("invalid context root: too complex")
		}
		web.ContextRoot = "/" + strings.Trim(rootContextURL.Path, "/") + "/"
	}
}

// SocketAddress returns the combination of the Address and the Port
func (web *webConfig) SocketAddress() string {
	return fmt.Sprintf("%s:%d", web.Address, web.Port)
}

// PrependWithContextRoot appends the given path to the ContextRoot
func (web *webConfig) PrependWithContextRoot(path string) string {
	return web.ContextRoot + strings.Trim(path, "/") + "/"
}
