package core

import (
	"bytes"
	"crypto/x509"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core/ui"
	"github.com/TwiN/gatus/v5/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	// importing the common health remove service across EG gRPC servers. 
	// Any servers built on top of the EG Stark SDK has this service out of the box.
	// If a target server is not built with the Stark SDK, the server should be updated to 
	// enable this optional standard health check service.
	// https://github.com/grpc/grpc/blob/master/doc/health-checking.md for details.
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type EndpointType string
type GrpcServiceNameToCheck string

const (
	// HostHeader is the name of the header used to specify the host
	HostHeader = "Host"

	// ContentTypeHeader is the name of the header used to specify the content type
	ContentTypeHeader = "Content-Type"

	// UserAgentHeader is the name of the header used to specify the request's user agent
	UserAgentHeader = "User-Agent"

	// GatusUserAgent is the default user agent that Gatus uses to send requests.
	GatusUserAgent = "Gatus/1.0"

	EndpointTypeDNS      EndpointType = "DNS"
	EndpointTypeTCP      EndpointType = "TCP"
	EndpointTypeSCTP     EndpointType = "SCTP"
	EndpointTypeUDP      EndpointType = "UDP"
	EndpointTypeICMP     EndpointType = "ICMP"
	EndpointTypeSTARTTLS EndpointType = "STARTTLS"
	EndpointTypeTLS      EndpointType = "TLS"
	EndpointTypeHTTP     EndpointType = "HTTP"
	EndpointTypeGRPC     EndpointType = "GRPC"
	EndpointTypeUNKNOWN  EndpointType = "UNKNOWN"
)

var (
	// ErrEndpointWithNoCondition is the error with which Gatus will panic if an endpoint is configured with no conditions
	ErrEndpointWithNoCondition = errors.New("you must specify at least one condition per endpoint")

	// ErrEndpointWithNoURL is the error with which Gatus will panic if an endpoint is configured with no url
	ErrEndpointWithNoURL = errors.New("you must specify an url for each endpoint")

	// ErrEndpointWithNoName is the error with which Gatus will panic if an endpoint is configured with no name
	ErrEndpointWithNoName = errors.New("you must specify a name for each endpoint")

	// ErrEndpointWithInvalidNameOrGroup is the error with which Gatus will panic if an endpoint has an invalid character where it shouldn't
	ErrEndpointWithInvalidNameOrGroup = errors.New("endpoint name and group must not have \" or \\")

	// ErrUnknownEndpointType is the error with which Gatus will panic if an endpoint has an unknown type
	ErrUnknownEndpointType = errors.New("unknown endpoint type")

	// ErrInvalidConditionFormat is the error with which Gatus will panic if a condition has an invalid format
	ErrInvalidConditionFormat = errors.New("invalid condition format: does not match '<VALUE> <COMPARATOR> <VALUE>'")

	// ErrInvalidEndpointIntervalForDomainExpirationPlaceholder is the error with which Gatus will panic if an endpoint
	// has both an interval smaller than 5 minutes and a condition with DomainExpirationPlaceholder.
	// This is because the free whois service we are using should not be abused, especially considering the fact that
	// the data takes a while to be updated.
	ErrInvalidEndpointIntervalForDomainExpirationPlaceholder = errors.New("the minimum interval for an endpoint with a condition using the " + DomainExpirationPlaceholder + " placeholder is 300s (5m)")

	// ErrInvalidGrpcServiceName is the error when a service name other than grpc.health.v[n].Health or unavailable service is passed.
	ErrInvalidGrpcServiceName = errors.New("Only grpc.health.v[n].Health remote service is supported: ")

	// ErrInvalidGrpcMethodName is the error when a method other than Check of the grpc.health.v[n].Health or unavailable method is passed.
	ErrInvalidGrpcMethodName = errors.New("Only Check method of the grpc.health.v[n].Health service is supported: ")

	// ErrInvalidGrpcHealthCheckRequestBody is the error when parsing is failed to extract the service name to check.
	ErrInvalidGrpcHealthCheckRequestBody = errors.New("Not able to find the service name: ")

	// ErrFailedToLoadCert is the error when a cert is read from a folder and loading the cert is failed. 
	ErrFailedToLoadCert = errors.New("Failed to load a server certificate in the configuration")

	// ErrInvalidGRPCUrl is the error when the GRPC URL is not in this format "//<hostname>:<port>" 
	ErrInvalidGRPCUrl = errors.New("Invalid GRPC URL string")
)

// Endpoint is the configuration of a monitored
type Endpoint struct {
	// Enabled defines whether to enable the monitoring of the endpoint
	Enabled *bool `yaml:"enabled,omitempty"`

	// Name of the endpoint. Can be anything.
	Name string `yaml:"name"`

	// Group the endpoint is a part of. Used for grouping multiple endpoints together on the front end.
	Group string `yaml:"group,omitempty"`

	// URL to send the request to
	URL string `yaml:"url"`

	// DNS is the configuration of DNS monitoring
	DNS *DNS `yaml:"dns,omitempty"`

	// Method of the request made to the url of the endpoint
	Method string `yaml:"method,omitempty"`

	// Body of the request
	Body string `yaml:"body,omitempty"`

	// GraphQL is whether to wrap the body in a query param ({"query":"$body"})
	GraphQL bool `yaml:"graphql,omitempty"`

	// GRPC used to wrap the RPC request with the service, method and body  
	GRPC Grpc `yaml:"grpc,omitempty"`

	// Headers of the request
	Headers map[string]string `yaml:"headers,omitempty"`

	// Interval is the duration to wait between every status check
	Interval time.Duration `yaml:"interval,omitempty"`

	// Conditions used to determine the health of the endpoint
	Conditions []Condition `yaml:"conditions"`

	// Alerts is the alerting configuration for the endpoint in case of failure
	Alerts []*alert.Alert `yaml:"alerts,omitempty"`

	// ClientConfig is the configuration of the client used to communicate with the endpoint's target
	ClientConfig *client.Config `yaml:"client,omitempty"`

	// UIConfig is the configuration for the UI
	UIConfig *ui.Config `yaml:"ui,omitempty"`

	// NumberOfFailuresInARow is the number of unsuccessful evaluations in a row
	NumberOfFailuresInARow int `yaml:"-"`

	// NumberOfSuccessesInARow is the number of successful evaluations in a row
	NumberOfSuccessesInARow int `yaml:"-"`
}

// IsEnabled returns whether the endpoint is enabled or not
func (endpoint Endpoint) IsEnabled() bool {
	if endpoint.Enabled == nil {
		return true
	}
	return *endpoint.Enabled
}

// Type returns the endpoint type
func (endpoint Endpoint) Type() EndpointType {
	switch {
	case endpoint.DNS != nil:
		return EndpointTypeDNS
	case strings.HasPrefix(endpoint.URL, "tcp://"):
		return EndpointTypeTCP
	case strings.HasPrefix(endpoint.URL, "sctp://"):
		return EndpointTypeSCTP
	case strings.HasPrefix(endpoint.URL, "udp://"):
		return EndpointTypeUDP
	case strings.HasPrefix(endpoint.URL, "icmp://"):
		return EndpointTypeICMP
	case strings.HasPrefix(endpoint.URL, "starttls://"):
		return EndpointTypeSTARTTLS
	case strings.HasPrefix(endpoint.URL, "tls://"):
		return EndpointTypeTLS
	case strings.HasPrefix(endpoint.URL, "http://") || strings.HasPrefix(endpoint.URL, "https://"):
		return EndpointTypeHTTP
	case strings.HasPrefix(endpoint.URL, "//") && len(endpoint.GRPC.Service) > 0:
		return EndpointTypeGRPC
	default:
		return EndpointTypeUNKNOWN
	}
}

// ValidateAndSetDefaults validates the endpoint's configuration and sets the default value of args that have one
func (endpoint *Endpoint) ValidateAndSetDefaults() error {
	// Set default values
	if endpoint.ClientConfig == nil {
		endpoint.ClientConfig = client.GetDefaultConfig()
	} else {
		if err := endpoint.ClientConfig.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	if endpoint.UIConfig == nil {
		endpoint.UIConfig = ui.GetDefaultConfig()
	} else {
		if err := endpoint.UIConfig.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	if endpoint.Interval == 0 {
		endpoint.Interval = 1 * time.Minute
	}
	if len(endpoint.Method) == 0 {
		endpoint.Method = http.MethodGet
	}
	if len(endpoint.Headers) == 0 {
		endpoint.Headers = make(map[string]string)
	}
	// Automatically add user agent header if there isn't one specified in the endpoint configuration
	if _, userAgentHeaderExists := endpoint.Headers[UserAgentHeader]; !userAgentHeaderExists {
		endpoint.Headers[UserAgentHeader] = GatusUserAgent
	}
	// Automatically add "Content-Type: application/json" header if there's no Content-Type set
	// and endpoint.GraphQL is set to true
	if _, contentTypeHeaderExists := endpoint.Headers[ContentTypeHeader]; !contentTypeHeaderExists && endpoint.GraphQL {
		endpoint.Headers[ContentTypeHeader] = "application/json"
	}
	for _, endpointAlert := range endpoint.Alerts {
		if err := endpointAlert.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	if len(endpoint.Name) == 0 {
		return ErrEndpointWithNoName
	}
	if strings.ContainsAny(endpoint.Name, "\"\\") || strings.ContainsAny(endpoint.Group, "\"\\") {
		return ErrEndpointWithInvalidNameOrGroup
	}
	if len(endpoint.URL) == 0 {
		return ErrEndpointWithNoURL
	}
	if len(endpoint.Conditions) == 0 {
		return ErrEndpointWithNoCondition
	}
	for _, c := range endpoint.Conditions {
		if endpoint.Interval < 5*time.Minute && c.hasDomainExpirationPlaceholder() {
			return ErrInvalidEndpointIntervalForDomainExpirationPlaceholder
		}
		if err := c.Validate(); err != nil {
			return fmt.Errorf("%v: %w", ErrInvalidConditionFormat, err)
		}
	}
	if endpoint.DNS != nil {
		return endpoint.DNS.validateAndSetDefault()
	}
	if endpoint.Type() == EndpointTypeUNKNOWN {
		return ErrUnknownEndpointType
	}
	// Make sure that the request can be created
	_, err := http.NewRequest(endpoint.Method, endpoint.URL, bytes.NewBuffer([]byte(endpoint.Body)))
	if err != nil {
		return err
	}
	return nil
}

// DisplayName returns an identifier made up of the Name and, if not empty, the Group.
func (endpoint Endpoint) DisplayName() string {
	if len(endpoint.Group) > 0 {
		return endpoint.Group + "/" + endpoint.Name
	}
	return endpoint.Name
}

// Key returns the unique key for the Endpoint
func (endpoint Endpoint) Key() string {
	return util.ConvertGroupAndEndpointNameToKey(endpoint.Group, endpoint.Name)
}

// EvaluateHealth sends a request to the endpoint's URL and evaluates the conditions of the endpoint.
func (endpoint *Endpoint) EvaluateHealth() *Result {
	result := &Result{Success: true, Errors: []string{}}
	// Parse or extract hostname from URL
	if endpoint.DNS != nil {
		result.Hostname = strings.TrimSuffix(endpoint.URL, ":53")
	} else if len(endpoint.GRPC.Service) != 0 {
		// sanitize endpoint.URL. GRPC url should start with "//" and both hostname and port should be 
		// passed in //<hostname>:<port> format.
		if !strings.HasPrefix(endpoint.URL, "//") {
			result.AddError(ErrInvalidGRPCUrl.Error())
		} else {
			urlObject, err := url.Parse(endpoint.URL)
			if err != nil {
				result.AddError(err.Error())
			} else {
				port := urlObject.Port()
				if len(port) == 0 {
					result.AddError(ErrInvalidGRPCUrl.Error())	
				}
				result.Hostname = urlObject.Hostname() 	// Prepended "//" is stripped
				endpoint.URL = result.Hostname+":"+port	// now GRPC's endpoint.URL shouldn't have prepended "//"
			}
		}
	} else {
		urlObject, err := url.Parse(endpoint.URL)
		if err != nil {
			result.AddError(err.Error())
		} else {
			result.Hostname = urlObject.Hostname()
		}
	}
	// Retrieve IP if necessary
	if endpoint.needsToRetrieveIP() {
		endpoint.getIP(result)
	}
	// Retrieve domain expiration if necessary
	if endpoint.needsToRetrieveDomainExpiration() && len(result.Hostname) > 0 {
		var err error
		if result.DomainExpiration, err = client.GetDomainExpiration(result.Hostname); err != nil {
			result.AddError(err.Error())
		}
	}
	// Call the endpoint (if there's no errors)
	if len(result.Errors) == 0 {
		endpoint.call(result)
	} else {
		result.Success = false
	}
	// Evaluate the conditions
	for _, condition := range endpoint.Conditions {
		success := condition.evaluate(result, endpoint.UIConfig.DontResolveFailedConditions)
		if !success {
			result.Success = false
		}
	}
	result.Timestamp = time.Now()
	// No need to keep the body after the endpoint has been evaluated
	result.body = nil
	// Clean up parameters that we don't need to keep in the results
	if endpoint.UIConfig.HideURL {
		for errIdx, errorString := range result.Errors {
			result.Errors[errIdx] = strings.ReplaceAll(errorString, endpoint.URL, "<redacted>")
		}
	}
	if endpoint.UIConfig.HideHostname {
		for errIdx, errorString := range result.Errors {
			result.Errors[errIdx] = strings.ReplaceAll(errorString, result.Hostname, "<redacted>")
		}
		result.Hostname = ""
	}
	return result
}

func (endpoint *Endpoint) getIP(result *Result) {
	if ips, err := net.LookupIP(result.Hostname); err != nil {
		result.AddError(err.Error())
		return
	} else {
		result.IP = ips[0].String()
	}
}

func (endpoint *Endpoint) call(result *Result) {
	var request *http.Request
	var response *http.Response
	var err error
	var certificate *x509.Certificate
	var serviceNameToCheck GrpcServiceNameToCheck

	endpointType := endpoint.Type()
	if endpointType == EndpointTypeHTTP {
		request = endpoint.buildHTTPRequest()
	} else if endpointType == EndpointTypeGRPC {
		serviceNameToCheck, err = endpoint.buildGRPCRequest()
		if err != nil {
			// just log and ignore the error so it will fail later during the time tracked. 
			// When failed user will be notified as configured in alert 
			log.Printf("%v", err)
		}
	}
	startTime := time.Now()

	if endpointType == EndpointTypeDNS {
		endpoint.DNS.query(endpoint.URL, result)
		result.Duration = time.Since(startTime)
	} else if endpointType == EndpointTypeSTARTTLS || endpointType == EndpointTypeTLS {
		if endpointType == EndpointTypeSTARTTLS {
			result.Connected, certificate, err = client.CanPerformStartTLS(strings.TrimPrefix(endpoint.URL, "starttls://"), endpoint.ClientConfig)
		} else {
			result.Connected, certificate, err = client.CanPerformTLS(strings.TrimPrefix(endpoint.URL, "tls://"), endpoint.ClientConfig)
		}
		if err != nil {
			result.AddError(err.Error())
			return
		}
		result.Duration = time.Since(startTime)
		result.CertificateExpiration = time.Until(certificate.NotAfter)
	} else if endpointType == EndpointTypeTCP {
		result.Connected = client.CanCreateTCPConnection(strings.TrimPrefix(endpoint.URL, "tcp://"), endpoint.ClientConfig)
		result.Duration = time.Since(startTime)
	} else if endpointType == EndpointTypeUDP {
		result.Connected = client.CanCreateUDPConnection(strings.TrimPrefix(endpoint.URL, "udp://"), endpoint.ClientConfig)
		result.Duration = time.Since(startTime)
	} else if endpointType == EndpointTypeSCTP {
		result.Connected = client.CanCreateSCTPConnection(strings.TrimPrefix(endpoint.URL, "sctp://"), endpoint.ClientConfig)
		result.Duration = time.Since(startTime)
	} else if endpointType == EndpointTypeICMP {
		result.Connected, result.Duration = client.Ping(strings.TrimPrefix(endpoint.URL, "icmp://"), endpoint.ClientConfig)
	} else if endpointType == EndpointTypeGRPC {
		var grpcResponse *healthpb.HealthCheckResponse

		/// HealthCheck Client
		conn, err := client.GetGRPCClientConnection(endpoint.ClientConfig, endpoint.URL)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		healthClient := healthpb.NewHealthClient(conn)
		// for gracelful connection close
		defer conn.Close()

		/// Outgoing Context
		ctx, cancel := context.WithTimeout(context.Background(), endpoint.ClientConfig.Timeout)
		if len(endpoint.Headers) > 0 {
			md := metadata.New(endpoint.Headers)
			ctx = metadata.NewOutgoingContext(ctx, md)	
		}
		// for graceful cancel
		defer cancel()

		/// Execute Check method of HealthClient
		var respHeaders metadata.MD
		request := &healthpb.HealthCheckRequest {
			Service: string(serviceNameToCheck),
		}
		grpcResponse, err = healthClient.Check(ctx, request, grpc.Header(&respHeaders))
		result.Duration = time.Since(startTime)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		
		// for now, response headers are not used for condition checks yet. maybe for future
		// log.Printf("Response Headers: %v\n", respHeaders)

		// TODO: result.CertificateExpiration for gRPC? 

		result.GRPCHealthStatus = grpcResponse.GetStatus()
		// https://pkg.go.dev/google.golang.org/grpc@v1.51.0/health/grpc_health_v1#HealthCheckResponse_ServingStatus for possible
		// return values (0 ~ 3 as int32)
		result.Connected = grpcResponse.GetStatus() >= 0

		// Only read the body if there's a condition that uses the BodyPlaceholder
		if endpoint.needsToReadBody() {
			respJson := protojson.Format(grpcResponse)
			result.body = []byte(respJson)
		}
	} else {
		response, err = client.GetHTTPClient(endpoint.ClientConfig).Do(request)
		result.Duration = time.Since(startTime)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		defer response.Body.Close()
		if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
			certificate = response.TLS.PeerCertificates[0]
			result.CertificateExpiration = time.Until(certificate.NotAfter)
		}
		result.HTTPStatus = response.StatusCode
		result.Connected = response.StatusCode > 0
		// Only read the body if there's a condition that uses the BodyPlaceholder
		if endpoint.needsToReadBody() {
			result.body, err = io.ReadAll(response.Body)
			if err != nil {
				result.AddError("error reading response body:" + err.Error())
			}
		}
	}
}

// It doesn't build a request but just pre-process params required when a GRPC client is created later.
// These should not be time tracked as they are just prep before making a gRPC request.
// Hence call this method before time is tracked. 
func (endpoint *Endpoint) buildGRPCRequest() (GrpcServiceNameToCheck, error) {
	// load the cert file
	if len(endpoint.ClientConfig.Cert) == 0 && len(endpoint.ClientConfig.CertPath) != 0 {
		// load a server certificate from local disk 
		serverCertPath := endpoint.ClientConfig.Cert
		serverCA, err := os.ReadFile(serverCertPath)
		if err != nil {
			return "", fmt.Errorf("%v: %w", ErrFailedToLoadCert, err)
		}
		endpoint.ClientConfig.Cert = string(serverCA)
	}

	// preprosess headers
	for _, v := range endpoint.Headers {
		// TODO check if the header value contains fuctions.
		
		fmt.Println(""+v)
	}

	// sanitize the the grpc request
	reg := regexp.MustCompile(`^grpc\.health\.v\d\.Health$`)
	if reg.FindStringIndex(endpoint.GRPC.Service) == nil {
		return "", fmt.Errorf("%v: %s", ErrInvalidGrpcServiceName, endpoint.GRPC.Service)
	}

	if endpoint.GRPC.Method != "Check" {
		return "", fmt.Errorf("%v: %s", ErrInvalidGrpcMethodName, endpoint.GRPC.Method)
	}

	// extract the service name to check
	// empty service name means to check the overall server status
	var serviceNameToCheck GrpcServiceNameToCheck = ""
	reg = regexp.MustCompile(`"service":\s*"?([^"]*)"?`)
	data := reg.FindSubmatch([]byte(endpoint.GRPC.Body))

	if data != nil {
		var sname string
		i := 0
		for _, one := range data {					
			if i == 1 {
				sname = string(one)
			}
			i++
		}
		sname = strings.TrimSpace(sname)
		log.Println("Service Name to Check:"+sname)
		serviceNameToCheck = GrpcServiceNameToCheck(sname)
	} else {
		return "", fmt.Errorf("%v: %s", ErrInvalidGrpcHealthCheckRequestBody, endpoint.GRPC.Body)
	}

	return serviceNameToCheck, nil
}

func (endpoint *Endpoint) buildHTTPRequest() *http.Request {
	var bodyBuffer *bytes.Buffer
	if endpoint.GraphQL {
		graphQlBody := map[string]string{
			"query": endpoint.Body,
		}
		body, _ := json.Marshal(graphQlBody)
		bodyBuffer = bytes.NewBuffer(body)
	} else {
		bodyBuffer = bytes.NewBuffer([]byte(endpoint.Body))
	}
	request, _ := http.NewRequest(endpoint.Method, endpoint.URL, bodyBuffer)
	for k, v := range endpoint.Headers {
		request.Header.Set(k, v)
		if k == HostHeader {
			request.Host = v
		}
	}
	return request
}

// needsToReadBody checks if there's any condition that requires the response body to be read
func (endpoint *Endpoint) needsToReadBody() bool {
	for _, condition := range endpoint.Conditions {
		if condition.hasBodyPlaceholder() {
			return true
		}
	}
	return false
}

// needsToRetrieveDomainExpiration checks if there's any condition that requires a whois query to be performed
func (endpoint *Endpoint) needsToRetrieveDomainExpiration() bool {
	for _, condition := range endpoint.Conditions {
		if condition.hasDomainExpirationPlaceholder() {
			return true
		}
	}
	return false
}

// needsToRetrieveIP checks if there's any condition that requires an IP lookup
func (endpoint *Endpoint) needsToRetrieveIP() bool {
	for _, condition := range endpoint.Conditions {
		if condition.hasIPPlaceholder() {
			return true
		}
	}
	return false
}
