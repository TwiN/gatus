package core

import (
	"fmt"
	"github.com/TwinProduction/gatus/pattern"
	"log"
	"strings"
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
