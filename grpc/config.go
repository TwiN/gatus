package grpc

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	validServiceNameRegex = `^grpc\.health\.v\d\.Health$`
	passedServiceNameRegex = `"service":\s*"?([^"]*)"?`

	defaultService = "grpc.health.v1.Health"
	defaultMethod = "Check"
	defaultBody = `{"service": ""}`
)

var (
	defaultConfig = Config {
		Service: defaultService,
		Method:  defaultMethod,
		Body:    defaultBody,
	}

	// ErrInvalidGrpcServiceName is the error when a service name other than grpc.health.v[n].Health or unavailable service is passed.
	ErrInvalidGrpcServiceName = errors.New("Only grpc.health.v[n].Health remote service is supported: ")

	// ErrInvalidGrpcMethodName is the error when a method other than Check of the grpc.health.v[n].Health or unavailable method is passed.
	ErrInvalidGrpcMethodName = errors.New("Only Check method of the grpc.health.v[n].Health service is supported: ")

	// ErrInvalidGrpcHealthCheckRequestBody is the error when parsing is failed to extract the service name to check.
	ErrInvalidGrpcHealthCheckRequestBody = errors.New("Not able to find the service name: ")
)

type GrpcServiceNameToCheck string

// GetDefaultConfig returns a copy of the default configuration
func GetDefaultConfig() *Config {
	cfg := defaultConfig
	return &cfg
}

// Grpc Config is the configuration of the GRPC request for the Health Checking Protocol. 
// https://github.com/grpc/grpc/blob/master/doc/health-checking.md for details.  
type Config struct {
	// Service defines the service name to call. Currently only grpc.health.v[n].Health is valid.
	Service string `yaml:"service"`

	// Method defines the remote method name of the service. Currently only "Check" is valid
	Method string `yaml:"method,omitempty"`
	
	// Body defines the request body of grpc.health.v[n].Health.Check method. 
	// The valid format is {"service": <service name to monitor>}. If an empty string ({"service": ""}) 
	// is passed for the service name, the overall health of the GRPC server will be minitored. 
	Body string `yaml:"body,omitempty"`

	ServiceNameToCheck GrpcServiceNameToCheck
}

// ValidateAndSetDefaults validates the grpc configuration and sets the default values if necessary
func (c *Config) ValidateAndSetDefaults() error {
	 // sanitize the the grpc request
	reg := regexp.MustCompile(validServiceNameRegex)
	if reg.FindStringIndex(c.Service) == nil {
		return fmt.Errorf("%v: %s", ErrInvalidGrpcServiceName, c.Service)
	}

	if len(c.Method) == 0 {
		c.Method = defaultMethod
	} else {
		if c.Method != defaultMethod {
			return fmt.Errorf("%v: %s", ErrInvalidGrpcMethodName, c.Method)
		}
	}

	if len(c.Body) == 0 {
		c.Body = defaultBody
		c.ServiceNameToCheck = ""	// Empty string means checking overall gRPC server health status
	} else {
		reg = regexp.MustCompile(passedServiceNameRegex)
		data := reg.FindSubmatch([]byte(c.Body))
		if data == nil {
			return fmt.Errorf("%v: %s", ErrInvalidGrpcHealthCheckRequestBody, c.Body)
		} else {
			var sname string
			i := 0
			for _, one := range data {					
				if i == 1 {
					sname = string(one)
				}
				i++
			}
			sname = strings.TrimSpace(sname)
			c.ServiceNameToCheck = GrpcServiceNameToCheck(sname)
		}
	}
	return nil
}
