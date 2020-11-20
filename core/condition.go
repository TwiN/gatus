package core

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/jsonpath"
	"github.com/TwinProduction/gatus/pattern"
)

const (
	// StatusPlaceholder is a placeholder for a HTTP status.
	//
	// Values that could replace the placeholder: 200, 404, 500, ...
	StatusPlaceholder = "[STATUS]"

	// IPPlaceholder is a placeholder for an IP.
	//
	// Values that could replace the placeholder: 127.0.0.1, 10.0.0.1, ...
	IPPlaceholder = "[IP]"

	// DNSRCodePlaceholder is a place holder for DNS_RCODE
	//
	// Values that could be NOERROR, FORMERR, SERVFAIL, NXDOMAIN, NOTIMP and REFUSED
	DNSRCodePlaceholder = "[DNS_RCODE]"

	// ResponseTimePlaceholder is a placeholder for the request response time, in milliseconds.
	//
	// Values that could replace the placeholder: 1, 500, 1000, ...
	ResponseTimePlaceholder = "[RESPONSE_TIME]"

	// BodyPlaceholder is a placeholder for the body of the response
	//
	// Values that could replace the placeholder: {}, {"data":{"name":"john"}}, ...
	BodyPlaceholder = "[BODY]"

	// ConnectedPlaceholder is a placeholder for whether a connection was successfully established.
	//
	// Values that could replace the placeholder: true, false
	ConnectedPlaceholder = "[CONNECTED]"

	// CertificateExpirationPlaceholder is a placeholder for the duration before certificate expiration, in milliseconds.
	//
	// Values that could replace the placeholder: 4461677039 (~52 days)
	CertificateExpirationPlaceholder = "[CERTIFICATE_EXPIRATION]"

	// LengthFunctionPrefix is the prefix for the length function
	LengthFunctionPrefix = "len("

	// PatternFunctionPrefix is the prefix for the pattern function
	PatternFunctionPrefix = "pat("

	// FunctionSuffix is the suffix for all functions
	FunctionSuffix = ")"

	// InvalidConditionElementSuffix is the suffix that will be appended to an invalid condition
	InvalidConditionElementSuffix = "(INVALID)"
)

// Condition is a condition that needs to be met in order for a Service to be considered healthy.
type Condition string

// evaluate the Condition with the Result of the health check
func (c *Condition) evaluate(result *Result) bool {
	condition := string(*c)
	success := false
	var resolvedCondition string
	if strings.Contains(condition, "==") {
		parts := sanitizeAndResolve(strings.Split(condition, "=="), result)
		success = isEqual(parts[0], parts[1])
		resolvedCondition = fmt.Sprintf("%v == %v", parts[0], parts[1])
	} else if strings.Contains(condition, "!=") {
		parts := sanitizeAndResolve(strings.Split(condition, "!="), result)
		success = !isEqual(parts[0], parts[1])
		resolvedCondition = fmt.Sprintf("%v != %v", parts[0], parts[1])
	} else if strings.Contains(condition, "<=") {
		parts := sanitizeAndResolveNumerical(strings.Split(condition, "<="), result)
		success = parts[0] <= parts[1]
		resolvedCondition = fmt.Sprintf("%v <= %v", parts[0], parts[1])
	} else if strings.Contains(condition, ">=") {
		parts := sanitizeAndResolveNumerical(strings.Split(condition, ">="), result)
		success = parts[0] >= parts[1]
		resolvedCondition = fmt.Sprintf("%v >= %v", parts[0], parts[1])
	} else if strings.Contains(condition, ">") {
		parts := sanitizeAndResolveNumerical(strings.Split(condition, ">"), result)
		success = parts[0] > parts[1]
		resolvedCondition = fmt.Sprintf("%v > %v", parts[0], parts[1])
	} else if strings.Contains(condition, "<") {
		parts := sanitizeAndResolveNumerical(strings.Split(condition, "<"), result)
		success = parts[0] < parts[1]
		resolvedCondition = fmt.Sprintf("%v < %v", parts[0], parts[1])
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid condition '%s' has been provided", condition))
		return false
	}
	conditionToDisplay := condition
	// If the condition isn't a success, return what the resolved condition was too
	if !success {
		log.Printf("[Condition][evaluate] Condition '%s' did not succeed because '%s' is false", condition, resolvedCondition)
		// Check if the resolved condition was an invalid path
		isResolvedConditionInvalidPath := strings.ReplaceAll(resolvedCondition, fmt.Sprintf("%s ", InvalidConditionElementSuffix), "") == condition
		if isResolvedConditionInvalidPath {
			// Since, in the event of an invalid path, the resolvedCondition contains the condition itself,
			// we'll only display the resolvedCondition
			conditionToDisplay = resolvedCondition
		} else {
			conditionToDisplay = fmt.Sprintf("%s (%s)", condition, resolvedCondition)
		}
	}
	result.ConditionResults = append(result.ConditionResults, &ConditionResult{Condition: conditionToDisplay, Success: success})
	return success
}

// isEqual compares two strings.
//
// It also supports the pattern function. That is to say, if one of the strings starts with PatternFunctionPrefix
// and ends with FunctionSuffix, it will be treated like a pattern.
func isEqual(first, second string) bool {
	var isFirstPattern, isSecondPattern bool
	if strings.HasPrefix(first, PatternFunctionPrefix) && strings.HasSuffix(first, FunctionSuffix) {
		isFirstPattern = true
		first = strings.TrimSuffix(strings.TrimPrefix(first, PatternFunctionPrefix), FunctionSuffix)
	}
	if strings.HasPrefix(second, PatternFunctionPrefix) && strings.HasSuffix(second, FunctionSuffix) {
		isSecondPattern = true
		second = strings.TrimSuffix(strings.TrimPrefix(second, PatternFunctionPrefix), FunctionSuffix)
	}
	if isFirstPattern && !isSecondPattern {
		return pattern.Match(first, second)
	} else if !isFirstPattern && isSecondPattern {
		return pattern.Match(second, first)
	} else {
		return first == second
	}
}

// sanitizeAndResolve sanitizes and resolves a list of element and returns the list of resolved elements
func sanitizeAndResolve(list []string, result *Result) []string {
	var sanitizedList []string
	body := strings.TrimSpace(string(result.Body))
	for _, element := range list {
		element = strings.TrimSpace(element)
		switch strings.ToUpper(element) {
		case StatusPlaceholder:
			element = strconv.Itoa(result.HTTPStatus)
		case IPPlaceholder:
			element = result.IP
		case ResponseTimePlaceholder:
			element = strconv.Itoa(int(result.Duration.Milliseconds()))
		case BodyPlaceholder:
			element = body
		case DNSRCodePlaceholder:
			element = result.DNSRCode
		case ConnectedPlaceholder:
			element = strconv.FormatBool(result.Connected)
		case CertificateExpirationPlaceholder:
			element = strconv.FormatInt(result.CertificateExpiration.Milliseconds(), 10)
		default:
			// if contains the BodyPlaceholder, then evaluate json path
			if strings.Contains(element, BodyPlaceholder) {
				wantLength := false
				if strings.HasPrefix(element, LengthFunctionPrefix) && strings.HasSuffix(element, FunctionSuffix) {
					wantLength = true
					element = strings.TrimSuffix(strings.TrimPrefix(element, LengthFunctionPrefix), FunctionSuffix)
				}
				resolvedElement, resolvedElementLength, err := jsonpath.Eval(strings.Replace(element, fmt.Sprintf("%s.", BodyPlaceholder), "", 1), result.Body)
				if err != nil {
					if err.Error() != "unexpected end of JSON input" {
						result.Errors = append(result.Errors, err.Error())
					}
					if wantLength {
						element = fmt.Sprintf("len(%s) %s", element, InvalidConditionElementSuffix)
					} else {
						element = fmt.Sprintf("%s %s", element, InvalidConditionElementSuffix)
					}
				} else {
					if wantLength {
						element = fmt.Sprintf("%d", resolvedElementLength)
					} else {
						element = resolvedElement
					}
				}
			}
		}
		sanitizedList = append(sanitizedList, element)
	}
	return sanitizedList
}

func sanitizeAndResolveNumerical(list []string, result *Result) []int64 {
	var sanitizedNumbers []int64
	sanitizedList := sanitizeAndResolve(list, result)
	for _, element := range sanitizedList {
		if duration, err := time.ParseDuration(element); duration != 0 && err == nil {
			sanitizedNumbers = append(sanitizedNumbers, duration.Milliseconds())
		} else if number, err := strconv.ParseInt(element, 10, 64); err != nil {
			// Default to 0 if the string couldn't be converted to an integer
			sanitizedNumbers = append(sanitizedNumbers, 0)
		} else {
			sanitizedNumbers = append(sanitizedNumbers, number)
		}
	}
	return sanitizedNumbers
}
