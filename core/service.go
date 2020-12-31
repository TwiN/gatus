package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/client"
)

const (
	// HostHeader is the name of the header used to specify the host
	HostHeader = "Host"

	// ContentTypeHeader is the name of the header used to specify the content type
	ContentTypeHeader = "Content-Type"

	// UserAgentHeader is the name of the header used to specify the request's user agent
	UserAgentHeader = "User-Agent"

	// GatusUserAgent is the default user agent that Gatus uses to send requests.
	GatusUserAgent = "Gatus/1.0"
)

var (
	// ErrServiceWithNoCondition is the error with which Gatus will panic if a service is configured with no conditions
	ErrServiceWithNoCondition = errors.New("you must specify at least one condition per service")

	// ErrServiceWithNoURL is the error with which Gatus will panic if a service is configured with no url
	ErrServiceWithNoURL = errors.New("you must specify an url for each service")

	// ErrServiceWithNoName is the error with which Gatus will panic if a service is configured with no name
	ErrServiceWithNoName = errors.New("you must specify a name for each service")
)

// Service is the configuration of a monitored endpoint
type Service struct {
	// Name of the service. Can be anything.
	Name string `yaml:"name"`

	// Group the service is a part of. Used for grouping multiple services together on the front end.
	Group string `yaml:"group,omitempty"`

	// URL to send the request to
	URL string `yaml:"url"`

	// DNS is the configuration of DNS monitoring
	DNS *DNS `yaml:"dns,omitempty"`

	// Method of the request made to the url of the service
	Method string `yaml:"method,omitempty"`

	// Body of the request
	Body string `yaml:"body,omitempty"`

	// GraphQL is whether to wrap the body in a query param ({"query":"$body"})
	GraphQL bool `yaml:"graphql,omitempty"`

	// Headers of the request
	Headers map[string]string `yaml:"headers,omitempty"`

	// Interval is the duration to wait between every status check
	Interval time.Duration `yaml:"interval,omitempty"`

	// Conditions used to determine the health of the service
	Conditions []*Condition `yaml:"conditions"`

	// Alerts is the alerting configuration for the service in case of failure
	Alerts []*Alert `yaml:"alerts"`

	// Insecure is whether to skip verifying the server's certificate chain and host name
	Insecure bool `yaml:"insecure,omitempty"`

	// NumberOfFailuresInARow is the number of unsuccessful evaluations in a row
	NumberOfFailuresInARow int

	// NumberOfFailuresInARow is the number of successful evaluations in a row
	NumberOfSuccessesInARow int
}

// ValidateAndSetDefaults validates the service's configuration and sets the default value of fields that have one
func (service *Service) ValidateAndSetDefaults() {
	// Set default values
	if service.Interval == 0 {
		service.Interval = 1 * time.Minute
	}
	if len(service.Method) == 0 {
		service.Method = http.MethodGet
	}
	if len(service.Headers) == 0 {
		service.Headers = make(map[string]string)
	}
	// Automatically add user agent header if there isn't one specified in the service configuration
	if _, userAgentHeaderExists := service.Headers[UserAgentHeader]; !userAgentHeaderExists {
		service.Headers[UserAgentHeader] = GatusUserAgent
	}
	// Automatically add "Content-Type: application/json" header if there's no Content-Type set
	// and service.GraphQL is set to true
	if _, contentTypeHeaderExists := service.Headers[ContentTypeHeader]; !contentTypeHeaderExists && service.GraphQL {
		service.Headers[ContentTypeHeader] = "application/json"
	}
	for _, alert := range service.Alerts {
		if alert.FailureThreshold <= 0 {
			alert.FailureThreshold = 3
		}
		if alert.SuccessThreshold <= 0 {
			alert.SuccessThreshold = 2
		}
	}
	if len(service.Name) == 0 {
		panic(ErrServiceWithNoName)
	}
	if len(service.URL) == 0 {
		panic(ErrServiceWithNoURL)
	}
	if len(service.Conditions) == 0 {
		panic(ErrServiceWithNoCondition)
	}
	if service.DNS != nil {
		service.DNS.validateAndSetDefault()
		return
	}
	// Make sure that the request can be created
	_, err := http.NewRequest(service.Method, service.URL, bytes.NewBuffer([]byte(service.Body)))
	if err != nil {
		panic(err)
	}
}

// EvaluateHealth sends a request to the service's URL and evaluates the conditions of the service.
func (service *Service) EvaluateHealth() *Result {
	result := &Result{Success: true, Errors: []string{}}
	service.getIP(result)
	if len(result.Errors) == 0 {
		service.call(result)
	} else {
		result.Success = false
	}
	for _, condition := range service.Conditions {
		success := condition.evaluate(result)
		if !success {
			result.Success = false
		}
	}
	result.Timestamp = time.Now()
	return result
}

// GetAlertsTriggered returns a slice of alerts that have been triggered
func (service *Service) GetAlertsTriggered() []Alert {
	var alerts []Alert
	if service.NumberOfFailuresInARow == 0 {
		return alerts
	}
	for _, alert := range service.Alerts {
		if alert.Enabled && alert.FailureThreshold == service.NumberOfFailuresInARow {
			alerts = append(alerts, *alert)
			continue
		}
	}
	return alerts
}

func (service *Service) getIP(result *Result) {
	if service.DNS != nil {
		result.Hostname = strings.TrimSuffix(service.URL, ":53")
	} else {
		urlObject, err := url.Parse(service.URL)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			return
		}
		result.Hostname = urlObject.Hostname()
	}
	ips, err := net.LookupIP(result.Hostname)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return
	}
	result.IP = ips[0].String()
}

func (service *Service) call(result *Result) {
	var request *http.Request
	var response *http.Response
	var err error
	isServiceDNS := service.DNS != nil
	isServiceTCP := strings.HasPrefix(service.URL, "tcp://")
	isServiceICMP := strings.HasPrefix(service.URL, "icmp://")
	isServiceHTTP := !isServiceDNS && !isServiceTCP && !isServiceICMP
	if isServiceHTTP {
		request = service.buildHTTPRequest()
	}
	startTime := time.Now()
	if isServiceDNS {
		service.DNS.query(service.URL, result)
		result.Duration = time.Since(startTime)
	} else if isServiceTCP {
		result.Connected = client.CanCreateTCPConnection(strings.TrimPrefix(service.URL, "tcp://"))
		result.Duration = time.Since(startTime)
	} else if isServiceICMP {
		result.Connected, result.Duration = client.Ping(strings.TrimPrefix(service.URL, "icmp://"))
	} else {
		response, err = client.GetHTTPClient(service.Insecure).Do(request)
		result.Duration = time.Since(startTime)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			return
		}
		if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
			certificate := response.TLS.PeerCertificates[0]
			result.CertificateExpiration = certificate.NotAfter.Sub(time.Now())
		}
		result.HTTPStatus = response.StatusCode
		result.Connected = response.StatusCode > 0
		result.Body, err = ioutil.ReadAll(response.Body)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}
}

func (service *Service) buildHTTPRequest() *http.Request {
	var bodyBuffer *bytes.Buffer
	if service.GraphQL {
		graphQlBody := map[string]string{
			"query": service.Body,
		}
		body, _ := json.Marshal(graphQlBody)
		bodyBuffer = bytes.NewBuffer(body)
	} else {
		bodyBuffer = bytes.NewBuffer([]byte(service.Body))
	}
	request, _ := http.NewRequest(service.Method, service.URL, bodyBuffer)
	for k, v := range service.Headers {
		request.Header.Set(k, v)
		if k == HostHeader {
			request.Host = v
		}
	}
	return request
}
