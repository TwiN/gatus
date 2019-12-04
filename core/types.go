package core

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
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
	Ip               string             `json:"ip"`
	Duration         time.Duration      `json:"duration"`
	Errors           []error            `json:"errors"`
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

func (service *Service) getStatus(result *Result) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	startTime := time.Now()
	response, err := client.Get(service.Url)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}
	result.Duration = time.Now().Sub(startTime)
	result.HttpStatus = response.StatusCode
}

func (service *Service) EvaluateConditions() *Result {
	result := &Result{Success: true}
	service.getStatus(result)
	service.getIp(result)
	if len(result.Errors) > 0 {
		result.Success = false
	}
	for _, condition := range service.Conditions {
		success := condition.Evaluate(result)
		if !success {
			result.Success = false
		}
	}
	result.Timestamp = time.Now()
	return result
}

type ConditionResult struct {
	Condition   *Condition `json:"condition"`
	Success     bool       `json:"success"`
	Explanation string     `json:"explanation"`
}

type Condition string

func (c *Condition) Evaluate(result *Result) bool {
	condition := string(*c)
	if strings.Contains(condition, "==") {
		parts := sanitizeAndResolve(strings.Split(condition, "=="), result)
		if parts[0] == parts[1] {
			result.ConditionResults = append(result.ConditionResults, &ConditionResult{
				Condition:   c,
				Success:     true,
				Explanation: fmt.Sprintf("%s is equal to %s", parts[0], parts[1]),
			})
			return true
		} else {
			result.ConditionResults = append(result.ConditionResults, &ConditionResult{
				Condition:   c,
				Success:     false,
				Explanation: fmt.Sprintf("%s is not equal to %s", parts[0], parts[1]),
			})
			return false
		}
	} else if strings.Contains(condition, "!=") {
		parts := sanitizeAndResolve(strings.Split(condition, "!="), result)
		if parts[0] != parts[1] {
			result.ConditionResults = append(result.ConditionResults, &ConditionResult{
				Condition:   c,
				Success:     true,
				Explanation: fmt.Sprintf("%s is not equal to %s", parts[0], parts[1]),
			})
			return true
		} else {
			result.ConditionResults = append(result.ConditionResults, &ConditionResult{
				Condition:   c,
				Success:     false,
				Explanation: fmt.Sprintf("%s is equal to %s", parts[0], parts[1]),
			})
			return false
		}
	} else {
		result.Errors = append(result.Errors, errors.New(fmt.Sprintf("invalid condition '%s' has been provided", condition)))
		return false
	}
}

func sanitizeAndResolve(list []string, result *Result) []string {
	var sanitizedList []string
	for _, element := range list {
		element = strings.TrimSpace(element)
		switch strings.ToUpper(element) {
		case "[STATUS]":
			element = strconv.Itoa(result.HttpStatus)
		case "[IP]":
			element = result.Ip
		default:
		}
		sanitizedList = append(sanitizedList, element)
	}
	return sanitizedList
}
