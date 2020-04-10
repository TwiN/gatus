package core

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	StatusPlaceholder       = "[STATUS]"
	IPPlaceHolder           = "[IP]"
	ResponseTimePlaceHolder = "[RESPONSE_TIME]"
)

type HealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ServerMessage struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

type Result struct {
	HttpStatus       int                `json:"status"`
	Hostname         string             `json:"hostname"`
	Ip               string             `json:"-"`
	Duration         time.Duration      `json:"duration"`
	Errors           []string           `json:"errors"`
	ConditionResults []*ConditionResult `json:"condition-results"`
	Success          bool               `json:"success"`
	Timestamp        time.Time          `json:"timestamp"`
}

type Service struct {
	Name       string        `yaml:"name"`
	Url        string        `yaml:"url"`
	Interval   time.Duration `yaml:"interval,omitempty"`
	Conditions []*Condition  `yaml:"conditions"`
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

func (service *Service) getStatus(result *Result) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	startTime := time.Now()
	response, err := client.Get(service.Url)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return
	}
	result.Duration = time.Now().Sub(startTime)
	result.HttpStatus = response.StatusCode
}

func (service *Service) EvaluateConditions() *Result {
	result := &Result{Success: true, Errors: []string{}}
	service.getIp(result)
	if len(result.Errors) == 0 {
		service.getStatus(result)
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

type ConditionResult struct {
	Condition *Condition `json:"condition"`
	Success   bool       `json:"success"`
}

type Condition string

func (c *Condition) evaluate(result *Result) bool {
	condition := string(*c)
	success := false
	if strings.Contains(condition, "==") {
		parts := sanitizeAndResolve(strings.Split(condition, "=="), result)
		success = parts[0] == parts[1]
	} else if strings.Contains(condition, "!=") {
		parts := sanitizeAndResolve(strings.Split(condition, "!="), result)
		success = parts[0] != parts[1]
	} else if strings.Contains(condition, "<=") {
		parts := sanitizeAndResolveNumerical(strings.Split(condition, "<="), result)
		success = parts[0] <= parts[1]
	} else if strings.Contains(condition, ">=") {
		parts := sanitizeAndResolveNumerical(strings.Split(condition, ">="), result)
		success = parts[0] >= parts[1]
	} else if strings.Contains(condition, ">") {
		parts := sanitizeAndResolveNumerical(strings.Split(condition, ">"), result)
		success = parts[0] > parts[1]
	} else if strings.Contains(condition, "<") {
		parts := sanitizeAndResolveNumerical(strings.Split(condition, "<"), result)
		success = parts[0] < parts[1]
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid condition '%s' has been provided", condition))
		return false
	}
	result.ConditionResults = append(result.ConditionResults, &ConditionResult{Condition: c, Success: success})
	return success
}

func sanitizeAndResolve(list []string, result *Result) []string {
	var sanitizedList []string
	for _, element := range list {
		element = strings.TrimSpace(element)
		switch strings.ToUpper(element) {
		case StatusPlaceholder:
			element = strconv.Itoa(result.HttpStatus)
		case IPPlaceHolder:
			element = result.Ip
		case ResponseTimePlaceHolder:
			element = strconv.Itoa(int(result.Duration.Milliseconds()))
		default:
		}
		sanitizedList = append(sanitizedList, element)
	}
	return sanitizedList
}

func sanitizeAndResolveNumerical(list []string, result *Result) []int {
	var sanitizedNumbers []int
	sanitizedList := sanitizeAndResolve(list, result)
	for _, element := range sanitizedList {
		if number, err := strconv.Atoi(element); err != nil {
			// Default to 0 if the string couldn't be converted to an integer
			sanitizedNumbers = append(sanitizedNumbers, 0)
		} else {
			sanitizedNumbers = append(sanitizedNumbers, number)
		}
	}
	return sanitizedNumbers
}
