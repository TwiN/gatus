package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/TwinProduction/gatus/client"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNoCondition = errors.New("you must specify at least one condition per service")
	ErrNoUrl       = errors.New("you must specify an url for each service")
)

// Service is the configuration of a monitored endpoint
type Service struct {
	// Name of the service. Can be anything.
	Name string `yaml:"name"`

	// URL to send the request to
	Url string `yaml:"url"`

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

	NumberOfFailuresInARow  int
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
	for _, alert := range service.Alerts {
		if alert.FailureThreshold <= 0 {
			alert.FailureThreshold = 3
		}
		if alert.SuccessThreshold <= 0 {
			alert.SuccessThreshold = 2
		}
	}
	if len(service.Url) == 0 {
		panic(ErrNoUrl)
	}
	if len(service.Conditions) == 0 {
		panic(ErrNoCondition)
	}

	// Make sure that the request can be created
	_, err := http.NewRequest(service.Method, service.Url, bytes.NewBuffer([]byte(service.Body)))
	if err != nil {
		panic(err)
	}
}

// EvaluateHealth sends a request to the service's URL and evaluates the conditions of the service.
func (service *Service) EvaluateHealth() *Result {
	result := &Result{Success: true, Errors: []string{}}
	service.getIp(result)
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

func (service *Service) getIp(result *Result) {
	urlObject, err := url.Parse(service.Url)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return
	}
	result.Hostname = urlObject.Hostname()
	ips, err := net.LookupIP(urlObject.Hostname())
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return
	}
	result.Ip = ips[0].String()
}

func (service *Service) call(result *Result) {
	isServiceTcp := strings.HasPrefix(service.Url, "tcp://")
	var request *http.Request
	var response *http.Response
	var err error
	if !isServiceTcp {
		request = service.buildRequest()
	}
	startTime := time.Now()
	if isServiceTcp {
		result.Connected = client.CanCreateConnectionToTcpService(strings.TrimPrefix(service.Url, "tcp://"))
		result.Duration = time.Since(startTime)
	} else {
		response, err = client.GetHttpClient(service.Insecure).Do(request)
		result.Duration = time.Since(startTime)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			return
		}
		result.HttpStatus = response.StatusCode
		result.Connected = response.StatusCode > 0
		result.Body, err = ioutil.ReadAll(response.Body)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}
}

func (service *Service) buildRequest() *http.Request {
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
	request, _ := http.NewRequest(service.Method, service.Url, bodyBuffer)
	for k, v := range service.Headers {
		request.Header.Set(k, v)
	}
	return request
}
