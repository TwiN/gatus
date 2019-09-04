package watchdog

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

type Request struct {
	Url string
}

type Result struct {
	HttpStatus int
	Hostname   string
	Ip         string
	Duration   time.Duration
	Errors     []error
}

func (request *Request) GetIp(result *Result) {
	urlObject, err := url.Parse(request.Url)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}
	result.Hostname = urlObject.Hostname()
	ips, err := net.LookupIP(urlObject.Hostname())
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}
	result.Ip = ips[0].String()
}

func (request *Request) GetStatus(result *Result) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	startTime := time.Now()
	response, err := client.Get(request.Url)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}
	result.Duration = time.Now().Sub(startTime)
	result.HttpStatus = response.StatusCode
}
