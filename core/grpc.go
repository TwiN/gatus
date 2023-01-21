package core

// Grpc is the configuration of the GRPC request for the Health Checking Service 
// (https://github.com/grpc/grpc/blob/master/doc/health-checking.md)  
type Grpc struct {
	// Service defines the service name to call. Currently only grpc.health.v[n].Health is valid.
	Service string `yaml:"service,omitempty"`

	// Method defines the remote method name of the service. Currently only "Check" is valid
	Method string `yaml:"method,omitempty"`
	
	// Body defines the request body of grpc.health.v[n].Health.Check method. 
	// The valid format is {"service": <service name to monitor>}. If an empty string ({"service": ""}) 
	// is passed for the service name, the overall health of the GRPC server will be minitored. 
	Body string `yaml:"body,omitempty"`
}
