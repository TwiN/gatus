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

	ContextRoot string `yaml:"context-root"`

	safeContextRoot string
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
		web.safeContextRoot = DefaultContextRoot
	} else {
		// url.PathEscape escapes all "/", in order to build a secure path
		// (1) split into path fragements using "/" as delimiter
		// (2) use url.PathEscape() on each fragment
		// (3) re-concatinate the path using "/" as join character
		const splitJoinChar = "/"
		pathes := strings.Split(web.ContextRoot, splitJoinChar)
		escapedPathes := make([]string, len(pathes))
		for i, path := range pathes {
			escapedPathes[i] = url.PathEscape(path)
		}

		web.safeContextRoot = strings.Join(escapedPathes, splitJoinChar)

		// assure that we have still a valid url
		_, err := url.Parse(web.safeContextRoot)
		if err != nil {
			panic(fmt.Sprintf("Invalid context root %s - Error %s", web.ContextRoot, err))
		}
	}
}

// SocketAddress returns the combination of the Address and the Port
func (web *webConfig) SocketAddress() string {
	return fmt.Sprintf("%s:%d", web.Address, web.Port)
}

// CtxRoot returns the context root
func (web *webConfig) CtxRoot() string {
	return web.safeContextRoot
}

// AppendToCtxRoot appends the given string to the context root
// AppendToCtxRoot takes care of having only one "/" character at
// the join point and exactly on "/" at the end
func (web *webConfig) AppendToCtxRoot(fragment string) string {
	return strings.TrimSuffix(web.safeContextRoot, "/") + "/" + strings.Trim(fragment, "/") + "/"
}
