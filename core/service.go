package core

import (
	"bytes"
	"github.com/TwinProduction/gatus/client"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Service struct {
	Name       string            `yaml:"name"`
	Interval   time.Duration     `yaml:"interval,omitempty"`
	Url        string            `yaml:"url"`
	Method     string            `yaml:"method,omitempty"`
	Body       string            `yaml:"body,omitempty"`
	Headers    map[string]string `yaml:"headers"`
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
	request, _ := http.NewRequest(service.Method, service.Url, bytes.NewBuffer([]byte(service.Body)))
	for k, v := range service.Headers {
		request.Header.Set(k, v)
	}
	return request
}
