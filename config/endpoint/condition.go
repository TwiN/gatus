package endpoint

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/config/gontext"
	"github.com/TwiN/gatus/v5/pattern"
)

const (
	// maximumLengthBeforeTruncatingWhenComparedWithPattern is the maximum length an element being compared to a
	// pattern can have.
	//
	// This is only used for aesthetic purposes; it does not influence whether the condition evaluation results in a
	// success or a failure
	maximumLengthBeforeTruncatingWhenComparedWithPattern = 25
)

// Condition is a condition that needs to be met in order for an Endpoint to be considered healthy.
type Condition string

// Validate checks if the Condition is valid
func (c Condition) Validate() error {
	r := &Result{}
	c.evaluate(r, false, false, nil)
	if len(r.Errors) != 0 {
		return errors.New(r.Errors[0])
	}
	return nil
}

// evaluate the Condition with the Result and an optional context
func (c Condition) evaluate(result *Result, dontResolveFailedConditions bool, resolveSuccessfulConditions bool, context *gontext.Gontext) bool {
	condition := string(c)
	success := false
	conditionToDisplay := condition
	shouldResolveCondition := func(success bool) bool {
		if success {
			return resolveSuccessfulConditions
		}
		return !dontResolveFailedConditions
	}
	if strings.Contains(condition, " == ") {
		parameters, resolvedParameters := sanitizeAndResolveWithContext(strings.Split(condition, " == "), result, context)
		success = isEqual(resolvedParameters[0], resolvedParameters[1])
		if shouldResolveCondition(success) {
			conditionToDisplay = prettify(parameters, resolvedParameters, "==")
		}
	} else if strings.Contains(condition, " != ") {
		parameters, resolvedParameters := sanitizeAndResolveWithContext(strings.Split(condition, " != "), result, context)
		success = !isEqual(resolvedParameters[0], resolvedParameters[1])
		if shouldResolveCondition(success) {
			conditionToDisplay = prettify(parameters, resolvedParameters, "!=")
		}
	} else if strings.Contains(condition, " <= ") {
		parameters, resolvedParameters := sanitizeAndResolveNumericalWithContext(strings.Split(condition, " <= "), result, context)
		success = resolvedParameters[0] <= resolvedParameters[1]
		if shouldResolveCondition(success) {
			conditionToDisplay = prettifyNumericalParameters(parameters, resolvedParameters, "<=")
		}
	} else if strings.Contains(condition, " >= ") {
		parameters, resolvedParameters := sanitizeAndResolveNumericalWithContext(strings.Split(condition, " >= "), result, context)
		success = resolvedParameters[0] >= resolvedParameters[1]
		if shouldResolveCondition(success) {
			conditionToDisplay = prettifyNumericalParameters(parameters, resolvedParameters, ">=")
		}
	} else if strings.Contains(condition, " > ") {
		parameters, resolvedParameters := sanitizeAndResolveNumericalWithContext(strings.Split(condition, " > "), result, context)
		success = resolvedParameters[0] > resolvedParameters[1]
		if shouldResolveCondition(success) {
			conditionToDisplay = prettifyNumericalParameters(parameters, resolvedParameters, ">")
		}
	} else if strings.Contains(condition, " < ") {
		parameters, resolvedParameters := sanitizeAndResolveNumericalWithContext(strings.Split(condition, " < "), result, context)
		success = resolvedParameters[0] < resolvedParameters[1]
		if shouldResolveCondition(success) {
			conditionToDisplay = prettifyNumericalParameters(parameters, resolvedParameters, "<")
		}
	} else {
		result.AddError(fmt.Sprintf("invalid condition: %s", condition))
		return false
	}
	if !success {
		//logr.Debugf("[Condition.evaluate] Condition '%s' did not succeed because '%s' is false", condition, condition)
	}
	result.ConditionResults = append(result.ConditionResults, &ConditionResult{Condition: conditionToDisplay, Success: success})
	return success
}

// hasBodyPlaceholder checks whether the condition has a BodyPlaceholder
// Used for determining whether the response body should be read or not
func (c Condition) hasBodyPlaceholder() bool {
	return strings.Contains(string(c), BodyPlaceholder)
}

// hasDomainExpirationPlaceholder checks whether the condition has a DomainExpirationPlaceholder
// Used for determining whether a whois operation is necessary
func (c Condition) hasDomainExpirationPlaceholder() bool {
	return strings.Contains(string(c), DomainExpirationPlaceholder)
}

// hasIPPlaceholder checks whether the condition has an IPPlaceholder
// Used for determining whether an IP lookup is necessary
func (c Condition) hasIPPlaceholder() bool {
	return strings.Contains(string(c), IPPlaceholder)
}

// isEqual compares two strings.
//
// Supports the "pat" and the "any" functions.
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

	// test if inputs are integers
	firstInt, err1 := strconv.ParseInt(first, 0, 64)
	secondInt, err2 := strconv.ParseInt(second, 0, 64)
	if err1 == nil && err2 == nil {
		return firstInt == secondInt
	}

	return first == second
}

// sanitizeAndResolveWithContext sanitizes and resolves a list of elements with an optional context
func sanitizeAndResolveWithContext(elements []string, result *Result, context *gontext.Gontext) ([]string, []string) {
	parameters := make([]string, len(elements))
	resolvedParameters := make([]string, len(elements))
	for i, element := range elements {
		element = strings.TrimSpace(element)
		parameters[i] = element

		// Use the unified ResolvePlaceholder function
		resolved, err := ResolvePlaceholder(element, result, context)
		if err != nil {
			// If there's an error, add it to the result
			result.AddError(err.Error())
			resolvedParameters[i] = element + " " + InvalidConditionElementSuffix
		} else {
			resolvedParameters[i] = resolved
		}
	}
	return parameters, resolvedParameters
}

func sanitizeAndResolveNumericalWithContext(list []string, result *Result, context *gontext.Gontext) (parameters []string, resolvedNumericalParameters []int64) {
	parameters, resolvedParameters := sanitizeAndResolveWithContext(list, result, context)
	for _, element := range resolvedParameters {
		if duration, err := time.ParseDuration(element); duration != 0 && err == nil {
			// If the string is a duration, convert it to milliseconds
			resolvedNumericalParameters = append(resolvedNumericalParameters, duration.Milliseconds())
		} else if number, err := strconv.ParseInt(element, 0, 64); err != nil {
			// It's not an int, so we'll check if it's a float
			if f, err := strconv.ParseFloat(element, 64); err == nil {
				// It's a float, but we'll convert it to an int. We're losing precision here, but it's better than
				// just returning 0.
				resolvedNumericalParameters = append(resolvedNumericalParameters, int64(f))
			} else {
				// Default to 0 if the string couldn't be converted to an integer or a float
				resolvedNumericalParameters = append(resolvedNumericalParameters, 0)
			}
		} else {
			resolvedNumericalParameters = append(resolvedNumericalParameters, number)
		}
	}
	return parameters, resolvedNumericalParameters
}

func prettifyNumericalParameters(parameters []string, resolvedParameters []int64, operator string) string {
	resolvedStrings := make([]string, 2)
	for i := 0; i < 2; i++ {
		// Check if the parameter is a certificate or domain expiration placeholder
		if parameters[i] == CertificateExpirationPlaceholder || parameters[i] == DomainExpirationPlaceholder {
			// Format as duration string (convert milliseconds back to duration)
			duration := time.Duration(resolvedParameters[i]) * time.Millisecond
			resolvedStrings[i] = formatDuration(duration)
		} else if _, err := time.ParseDuration(parameters[i]); err == nil {
			// If the original parameter was a duration string (like "48h"), format the resolved value
			// as a duration string too so it matches and doesn't show in parentheses
			duration := time.Duration(resolvedParameters[i]) * time.Millisecond
			resolvedStrings[i] = formatDuration(duration)
		} else {
			// Format as integer
			resolvedStrings[i] = strconv.Itoa(int(resolvedParameters[i]))
		}
	}
	return prettify(parameters, resolvedStrings, operator)
}

// formatDuration formats a duration in a clean, human-readable way by removing unnecessary zero components.
// For example: 336h0m0s becomes 336h, 1h30m0s becomes 1h30m, but 1h0m15s stays as 1h0m15s.
// Truncates to whole seconds to avoid decimal values like 7353h5m54.67s.
func formatDuration(d time.Duration) string {
	// Truncate to whole seconds to avoid decimal seconds
	d = d.Truncate(time.Second)
	s := d.String()
	// Special case: if duration is zero, return "0s"
	if s == "0s" {
		return "0s"
	}
	// Remove trailing "0s" if present
	if strings.HasSuffix(s, "0s") {
		s = strings.TrimSuffix(s, "0s")
		// Remove trailing "0m" if present after removing "0s"
		s = strings.TrimSuffix(s, "0m")
	}
	return s
}

// prettify returns a string representation of a condition with its parameters resolved between parentheses
func prettify(parameters []string, resolvedParameters []string, operator string) string {
	// Handle pattern function truncation first
	if strings.HasPrefix(parameters[0], PatternFunctionPrefix) && strings.HasSuffix(parameters[0], FunctionSuffix) && len(resolvedParameters[1]) > maximumLengthBeforeTruncatingWhenComparedWithPattern {
		resolvedParameters[1] = fmt.Sprintf("%.25s...(truncated)", resolvedParameters[1])
	}
	if strings.HasPrefix(parameters[1], PatternFunctionPrefix) && strings.HasSuffix(parameters[1], FunctionSuffix) && len(resolvedParameters[0]) > maximumLengthBeforeTruncatingWhenComparedWithPattern {
		resolvedParameters[0] = fmt.Sprintf("%.25s...(truncated)", resolvedParameters[0])
	}
	// Determine the state of each parameter
	leftChanged := parameters[0] != resolvedParameters[0]
	rightChanged := parameters[1] != resolvedParameters[1]
	leftInvalid := resolvedParameters[0] == parameters[0]+" "+InvalidConditionElementSuffix
	rightInvalid := resolvedParameters[1] == parameters[1]+" "+InvalidConditionElementSuffix
	// Build the output based on what was resolved
	var left, right string
	// Format left side
	if leftChanged && !leftInvalid {
		left = parameters[0] + " (" + resolvedParameters[0] + ")"
	} else if leftInvalid {
		left = resolvedParameters[0] // Already has (INVALID)
	} else {
		left = parameters[0] // Unchanged
	}
	// Format right side
	if rightChanged && !rightInvalid {
		right = parameters[1] + " (" + resolvedParameters[1] + ")"
	} else if rightInvalid {
		right = resolvedParameters[1] // Already has (INVALID)
	} else {
		right = parameters[1] // Unchanged
	}
	return left + " " + operator + " " + right
}
