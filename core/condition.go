package core

import (
	"fmt"
	"log"
	"strings"
)

type Condition string

func (c *Condition) evaluate(result *Result) bool {
	condition := string(*c)
	success := false
	var resolvedCondition string
	if strings.Contains(condition, "==") {
		parts := sanitizeAndResolve(strings.Split(condition, "=="), result)
		success = parts[0] == parts[1]
		resolvedCondition = fmt.Sprintf("%v == %v", parts[0], parts[1])
	} else if strings.Contains(condition, "!=") {
		parts := sanitizeAndResolve(strings.Split(condition, "!="), result)
		success = parts[0] != parts[1]
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
	// If the condition isn't a success, return the resolved condition
	if !success {
		log.Printf("[Condition][evaluate] Condition '%s' did not succeed because '%s' is false", condition, resolvedCondition)
		conditionToDisplay = resolvedCondition
	}
	result.ConditionResults = append(result.ConditionResults, &ConditionResult{Condition: conditionToDisplay, Success: success})
	return success
}
