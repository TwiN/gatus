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
	"time"
)

var (
	ErrNoCondition = errors.New("you must specify at least one condition per service")
	ErrNoUrl       = errors.New("you must specify an url for each service")
)

type Service struct {
	Name       string            `yaml:"name"`
	Url        string            `yaml:"url"`
	Method     string            `yaml:"method,omitempty"`
	Body       string            `yaml:"body,omitempty"`
	GraphQL    bool              `yaml:"graphql,omitempty"`
	Headers    map[string]string `yaml:"headers,omitempty"`
	Interval   time.Duration     `yaml:"interval,omitempty"`
	Conditions []*Condition      `yaml:"conditions"`
}

func (service *Service) Validate() {
	// Set default values
	if service.Interval == 0 {
		service.Interval = 10 * time.Second
	}
	if len(service.Method) == 0 {
		service.Method = http.MethodGet
	}
	if len(service.Headers) == 0 {
		service.Headers = make(map[string]string)
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

func (service *Service) EvaluateConditions() *Result {
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
	request := service.buildRequest()
	startTime := time.Now()
	response, err := client.GetHttpClient().Do(request)
	if err != nil {
		result.Duration = time.Since(startTime)
		result.Errors = append(result.Errors, err.Error())
		return
	}
	result.Duration = time.Since(startTime)
	result.HttpStatus = response.StatusCode
	result.Body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
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
