package core

import (
	"fmt"
	"github.com/TwinProduction/gatus/jsonpath"
	"github.com/TwinProduction/gatus/pattern"
	"log"
	"strconv"
	"strings"
)

const (
	StatusPlaceholder       = "[STATUS]"
	IPPlaceHolder           = "[IP]"
	ResponseTimePlaceHolder = "[RESPONSE_TIME]"
	BodyPlaceHolder         = "[BODY]"
	ConnectedPlaceHolder    = "[CONNECTED]"

	LengthFunctionPrefix  = "len("
	PatternFunctionPrefix = "pat("
	FunctionSuffix        = ")"

	InvalidConditionElementSuffix = "(INVALID)"
)

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
		conditionToDisplay = fmt.Sprintf("%s (%s)", condition, resolvedCondition)
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
			element = strconv.Itoa(result.HttpStatus)
		case IPPlaceHolder:
			element = result.Ip
		case ResponseTimePlaceHolder:
			element = strconv.Itoa(int(result.Duration.Milliseconds()))
		case BodyPlaceHolder:
			element = body
		case ConnectedPlaceHolder:
			element = strconv.FormatBool(result.Connected)
		default:
			// if contains the BodyPlaceHolder, then evaluate json path
			if strings.Contains(element, BodyPlaceHolder) {
				wantLength := false
				if strings.HasPrefix(element, LengthFunctionPrefix) && strings.HasSuffix(element, FunctionSuffix) {
					wantLength = true
					element = strings.TrimSuffix(strings.TrimPrefix(element, LengthFunctionPrefix), FunctionSuffix)
				}
				resolvedElement, resolvedElementLength, err := jsonpath.Eval(strings.Replace(element, fmt.Sprintf("%s.", BodyPlaceHolder), "", 1), result.Body)
				if err != nil {
					result.Errors = append(result.Errors, err.Error())
					element = fmt.Sprintf("%s %s", element, InvalidConditionElementSuffix)
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
