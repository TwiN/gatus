package core

import (
	"fmt"
	"github.com/TwinProduction/gatus/jsonpath"
	"strconv"
	"strings"
)

const (
	StatusPlaceholder       = "[STATUS]"
	IPPlaceHolder           = "[IP]"
	ResponseTimePlaceHolder = "[RESPONSE_TIME]"
	BodyPlaceHolder         = "[BODY]"

	InvalidConditionElementSuffix = "(INVALID)"
)

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
		default:
			// if starts with BodyPlaceHolder, then evaluate json path
			if strings.Contains(element, BodyPlaceHolder) {
				wantLength := false
				if strings.HasPrefix(element, "len(") && strings.HasSuffix(element, ")") {
					wantLength = true
					element = strings.TrimSuffix(strings.TrimPrefix(element, "len("), ")")
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
