package core

import (
	"fmt"
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
	//
	// Usage: len([BODY].articles) == 10, len([BODY].name) > 5
	LengthFunctionPrefix = "len("

	// PatternFunctionPrefix is the prefix for the pattern function
	//
	// Usage: pat(192.168.*.*)
	PatternFunctionPrefix = "pat("

	// AnyFunctionPrefix is the prefix for the any function
	//
	// Usage: any(1.1.1.1, 1.0.0.1)
	AnyFunctionPrefix = "any("

	// FunctionSuffix is the suffix for all functions
	FunctionSuffix = ")"

	// InvalidConditionElementSuffix is the suffix that will be appended to an invalid condition
	InvalidConditionElementSuffix = "(INVALID)"
)

// Condition is a condition that needs to be met in order for a Service to be considered healthy.
type Condition string

// evaluate the Condition with the Result of the health check
func (c Condition) evaluate(result *Result) bool {
	condition := string(c)
	success := false
	conditionToDisplay := condition
	if strings.Contains(condition, "==") {
		parameters, resolvedParameters := sanitizeAndResolve(strings.Split(condition, "=="), result)
		success = isEqual(resolvedParameters[0], resolvedParameters[1])
		if !success {
			conditionToDisplay = prettify(parameters, resolvedParameters, "==")
		}
	} else if strings.Contains(condition, "!=") {
		parameters, resolvedParameters := sanitizeAndResolve(strings.Split(condition, "!="), result)
		success = !isEqual(resolvedParameters[0], resolvedParameters[1])
		if !success {
			conditionToDisplay = prettify(parameters, resolvedParameters, "!=")
		}
	} else if strings.Contains(condition, "<=") {
		parameters, resolvedParameters := sanitizeAndResolveNumerical(strings.Split(condition, "<="), result)
		success = resolvedParameters[0] <= resolvedParameters[1]
		if !success {
			conditionToDisplay = prettifyNumericalParameters(parameters, resolvedParameters, "<=")
		}
	} else if strings.Contains(condition, ">=") {
		parameters, resolvedParameters := sanitizeAndResolveNumerical(strings.Split(condition, ">="), result)
		success = resolvedParameters[0] >= resolvedParameters[1]
		if !success {
			conditionToDisplay = prettifyNumericalParameters(parameters, resolvedParameters, ">=")
		}
	} else if strings.Contains(condition, ">") {
		parameters, resolvedParameters := sanitizeAndResolveNumerical(strings.Split(condition, ">"), result)
		success = resolvedParameters[0] > resolvedParameters[1]
		if !success {
			conditionToDisplay = prettifyNumericalParameters(parameters, resolvedParameters, ">")
		}
	} else if strings.Contains(condition, "<") {
		parameters, resolvedParameters := sanitizeAndResolveNumerical(strings.Split(condition, "<"), result)
		success = resolvedParameters[0] < resolvedParameters[1]
		if !success {
			conditionToDisplay = prettifyNumericalParameters(parameters, resolvedParameters, "<")
		}
	} else {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid condition '%s' has been provided", condition))
		return false
	}
	if !success {
		//log.Printf("[Condition][evaluate] Condition '%s' did not succeed because '%s' is false", condition, condition)
	}
	result.ConditionResults = append(result.ConditionResults, &ConditionResult{Condition: conditionToDisplay, Success: success})
	return success
}

// isEqual compares two strings.
//
// Supports the pattern and the any functions.
// i.e. if one of the parameters starts with PatternFunctionPrefix and ends with FunctionSuffix, it will be treated like
// a pattern.
func isEqual(first, second string) bool {
	firstHasFunctionSuffix := strings.HasSuffix(first, FunctionSuffix)
	secondHasFunctionSuffix := strings.HasSuffix(second, FunctionSuffix)
	if firstHasFunctionSuffix || secondHasFunctionSuffix {
		var isFirstPattern, isSecondPattern bool
		if strings.HasPrefix(first, PatternFunctionPrefix) && firstHasFunctionSuffix {
			isFirstPattern = true
			first = strings.TrimSuffix(strings.TrimPrefix(first, PatternFunctionPrefix), FunctionSuffix)
		}
		if strings.HasPrefix(second, PatternFunctionPrefix) && secondHasFunctionSuffix {
			isSecondPattern = true
			second = strings.TrimSuffix(strings.TrimPrefix(second, PatternFunctionPrefix), FunctionSuffix)
		}
		if isFirstPattern && !isSecondPattern {
			return pattern.Match(first, second)
		} else if !isFirstPattern && isSecondPattern {
			return pattern.Match(second, first)
		}
		var isFirstAny, isSecondAny bool
		if strings.HasPrefix(first, AnyFunctionPrefix) && firstHasFunctionSuffix {
			isFirstAny = true
			first = strings.TrimSuffix(strings.TrimPrefix(first, AnyFunctionPrefix), FunctionSuffix)
		}
		if strings.HasPrefix(second, AnyFunctionPrefix) && secondHasFunctionSuffix {
			isSecondAny = true
			second = strings.TrimSuffix(strings.TrimPrefix(second, AnyFunctionPrefix), FunctionSuffix)
		}
		if isFirstAny && !isSecondAny {
			options := strings.Split(first, ",")
			for _, option := range options {
				if strings.TrimSpace(option) == second {
					return true
				}
			}
			return false
		} else if !isFirstAny && isSecondAny {
			options := strings.Split(second, ",")
			for _, option := range options {
				if strings.TrimSpace(option) == first {
					return true
				}
			}
			return false
		}
	}
	return first == second
}

// sanitizeAndResolve sanitizes and resolves a list of elements and returns the list of parameters as well as a list
// of resolved parameters
func sanitizeAndResolve(elements []string, result *Result) ([]string, []string) {
	parameters := make([]string, len(elements))
	resolvedParameters := make([]string, len(elements))
	body := strings.TrimSpace(string(result.Body))
	for i, element := range elements {
		element = strings.TrimSpace(element)
		parameters[i] = element
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
				resolvedElement, resolvedElementLength, err := jsonpath.Eval(strings.TrimPrefix(element, BodyPlaceholder+"."), result.Body)
				if err != nil {
					if err.Error() != "unexpected end of JSON input" {
						result.Errors = append(result.Errors, err.Error())
					}
					if wantLength {
						element = LengthFunctionPrefix + element + FunctionSuffix + " " + InvalidConditionElementSuffix
					} else {
						element = element + " " + InvalidConditionElementSuffix
					}
				} else {
					if wantLength {
						element = strconv.Itoa(resolvedElementLength)
					} else {
						element = resolvedElement
					}
				}
			}
		}
		resolvedParameters[i] = element
	}
	return parameters, resolvedParameters
}

func sanitizeAndResolveNumerical(list []string, result *Result) (parameters []string, resolvedNumericalParameters []int64) {
	parameters, resolvedParameters := sanitizeAndResolve(list, result)
	for _, element := range resolvedParameters {
		if duration, err := time.ParseDuration(element); duration != 0 && err == nil {
			resolvedNumericalParameters = append(resolvedNumericalParameters, duration.Milliseconds())
		} else if number, err := strconv.ParseInt(element, 10, 64); err != nil {
			// Default to 0 if the string couldn't be converted to an integer
			resolvedNumericalParameters = append(resolvedNumericalParameters, 0)
		} else {
			resolvedNumericalParameters = append(resolvedNumericalParameters, number)
		}
	}
	return parameters, resolvedNumericalParameters
}

func prettifyNumericalParameters(parameters []string, resolvedParameters []int64, operator string) string {
	return prettify(parameters, []string{strconv.Itoa(int(resolvedParameters[0])), strconv.Itoa(int(resolvedParameters[1]))}, operator)
}

// XXX: make this configurable? i.e. show-resolved-conditions-on-failure
func prettify(parameters []string, resolvedParameters []string, operator string) string {
	// Since, in the event of an invalid path, the resolvedParameters also contain the condition itself,
	// we'll return the resolvedParameters as-is.
	if strings.HasSuffix(resolvedParameters[0], InvalidConditionElementSuffix) || strings.HasSuffix(resolvedParameters[1], InvalidConditionElementSuffix) {
		return resolvedParameters[0] + " " + operator + " " + resolvedParameters[1]
	}
	// First element is a placeholder
	if parameters[0] != resolvedParameters[0] && parameters[1] == resolvedParameters[1] {
		return parameters[0] + " (" + resolvedParameters[0] + ") " + operator + " " + parameters[1]
	}
	// Second element is a placeholder
	if parameters[0] == resolvedParameters[0] && parameters[1] != resolvedParameters[1] {
		return parameters[0] + " " + operator + " " + parameters[1] + " (" + resolvedParameters[1] + ")"
	}
	// Both elements are placeholders...?
	if parameters[0] != resolvedParameters[0] && parameters[1] != resolvedParameters[1] {
		return parameters[0] + " (" + resolvedParameters[0] + ") " + operator + " " + parameters[1] + " (" + resolvedParameters[1] + ")"
	}
	// Neither elements are placeholders
	return parameters[0] + " " + operator + " " + parameters[1]
}
