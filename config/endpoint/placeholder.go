package endpoint

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TwiN/gatus/v5/config/gontext"
	"github.com/TwiN/gatus/v5/jsonpath"
)

// Placeholders
const (
	// StatusPlaceholder is a placeholder for a HTTP status.
	//
	// Values that could replace the placeholder: 200, 404, 500, ...
	StatusPlaceholder = "[STATUS]"

	// IPPlaceholder is a placeholder for an IP.
	//
	// Values that could replace the placeholder: 127.0.0.1, 10.0.0.1, ...
	IPPlaceholder = "[IP]"

	// DNSRCodePlaceholder is a placeholder for DNS_RCODE
	//
	// Values that could replace the placeholder: NOERROR, FORMERR, SERVFAIL, NXDOMAIN, NOTIMP, REFUSED
	DNSRCodePlaceholder = "[DNS_RCODE]"

	// ResponseTimePlaceholder is a placeholder for the request response time, in milliseconds.
	//
	// Values that could replace the placeholder: 1, 500, 1000, ...
	ResponseTimePlaceholder = "[RESPONSE_TIME]"

	// BodyPlaceholder is a placeholder for the Body of the response
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

	// DomainExpirationPlaceholder is a placeholder for the duration before the domain expires, in milliseconds.
	DomainExpirationPlaceholder = "[DOMAIN_EXPIRATION]"

	// ContextPlaceholder is a placeholder for suite context values
	// Usage: [CONTEXT].path.to.value
	ContextPlaceholder = "[CONTEXT]"
)

// Functions
const (
	// LengthFunctionPrefix is the prefix for the length function
	//
	// Usage: len([BODY].articles) == 10, len([BODY].name) > 5
	LengthFunctionPrefix = "len("

	// HasFunctionPrefix is the prefix for the has function
	//
	// Usage: has([BODY].errors) == true
	HasFunctionPrefix = "has("

	// PatternFunctionPrefix is the prefix for the pattern function
	//
	// Usage: [IP] == pat(192.168.*.*)
	PatternFunctionPrefix = "pat("

	// AnyFunctionPrefix is the prefix for the any function
	//
	// Usage: [IP] == any(1.1.1.1, 1.0.0.1)
	AnyFunctionPrefix = "any("

	// FunctionSuffix is the suffix for all functions
	FunctionSuffix = ")"
)

// Other constants
const (
	// InvalidConditionElementSuffix is the suffix that will be appended to an invalid condition
	InvalidConditionElementSuffix = "(INVALID)"
)

// functionType represents the type of function wrapper
type functionType int

const (
	// Note that not all functions are handled here. Only len() and has() directly impact the handler
	// e.g. "len([BODY].name) > 0" vs pat() or any(), which would be used like "[BODY].name == pat(john*)"

	noFunction functionType = iota
	functionLen
	functionHas
)

// ResolvePlaceholder resolves all types of placeholders to their string values.
//
// Supported placeholders:
//   - [STATUS]: HTTP status code (e.g., "200", "404")
//   - [IP]: IP address from the response (e.g., "127.0.0.1")
//   - [RESPONSE_TIME]: Response time in milliseconds (e.g., "250")
//   - [DNS_RCODE]: DNS response code (e.g., "NOERROR", "NXDOMAIN")
//   - [CONNECTED]: Connection status (e.g., "true", "false")
//   - [CERTIFICATE_EXPIRATION]: Certificate expiration time in milliseconds
//   - [DOMAIN_EXPIRATION]: Domain expiration time in milliseconds
//   - [BODY]: Full response body
//   - [BODY].path: JSONPath expression on response body (e.g., [BODY].status, [BODY].data[0].name)
//   - [CONTEXT].path: Suite context values (e.g., [CONTEXT].user_id, [CONTEXT].session_token)
//
// Function wrappers:
//   - len(placeholder): Returns the length of the resolved value
//   - has(placeholder): Returns "true" if the placeholder exists and is non-empty, "false" otherwise
//
// Examples:
//   - ResolvePlaceholder("[STATUS]", result, nil) → "200"
//   - ResolvePlaceholder("len([BODY].items)", result, nil) → "5" (for JSON array with 5 items)
//   - ResolvePlaceholder("has([CONTEXT].user_id)", result, ctx) → "true" (if context has user_id)
//   - ResolvePlaceholder("[BODY].user.name", result, nil) → "john" (for {"user":{"name":"john"}})
//
// Case-insensitive: All placeholder names are handled case-insensitively, but paths preserve original case.
func ResolvePlaceholder(placeholder string, result *Result, ctx *gontext.Gontext) (string, error) {
	placeholder = strings.TrimSpace(placeholder)
	originalPlaceholder := placeholder

	// Extract function wrapper if present
	fn, innerPlaceholder := extractFunctionWrapper(placeholder)
	placeholder = innerPlaceholder

	// Handle CONTEXT placeholders
	uppercasePlaceholder := strings.ToUpper(placeholder)
	if strings.HasPrefix(uppercasePlaceholder, ContextPlaceholder) && ctx != nil {
		return resolveContextPlaceholder(placeholder, fn, originalPlaceholder, ctx)
	}

	// Handle basic placeholders (try uppercase first for backward compatibility)
	switch uppercasePlaceholder {
	case StatusPlaceholder:
		return formatWithFunction(strconv.Itoa(result.HTTPStatus), fn), nil
	case IPPlaceholder:
		return formatWithFunction(result.IP, fn), nil
	case ResponseTimePlaceholder:
		return formatWithFunction(strconv.FormatInt(result.Duration.Milliseconds(), 10), fn), nil
	case DNSRCodePlaceholder:
		return formatWithFunction(result.DNSRCode, fn), nil
	case ConnectedPlaceholder:
		return formatWithFunction(strconv.FormatBool(result.Connected), fn), nil
	case CertificateExpirationPlaceholder:
		return formatWithFunction(strconv.FormatInt(result.CertificateExpiration.Milliseconds(), 10), fn), nil
	case DomainExpirationPlaceholder:
		return formatWithFunction(strconv.FormatInt(result.DomainExpiration.Milliseconds(), 10), fn), nil
	case BodyPlaceholder:
		body := strings.TrimSpace(string(result.Body))
		if fn == functionHas {
			return strconv.FormatBool(len(body) > 0), nil
		}
		if fn == functionLen {
			// For len([BODY]), we need to check if it's JSON and get the actual length
			// Use jsonpath to evaluate the root element
			_, resolvedLength, err := jsonpath.Eval("", result.Body)
			if err == nil {
				return strconv.Itoa(resolvedLength), nil
			}
			// Fall back to string length if not valid JSON
			return strconv.Itoa(len(body)), nil
		}
		return body, nil
	}

	// Handle JSONPath expressions on BODY (including array indexing)
	if strings.HasPrefix(uppercasePlaceholder, BodyPlaceholder+".") || strings.HasPrefix(uppercasePlaceholder, BodyPlaceholder+"[") {
		return resolveJSONPathPlaceholder(placeholder, fn, originalPlaceholder, result)
	}

	// Not a recognized placeholder
	if fn != noFunction {
		if fn == functionHas {
			return "false", nil
		}
		// For len() with unrecognized placeholder, return with INVALID suffix
		return originalPlaceholder + " " + InvalidConditionElementSuffix, nil
	}

	// Return the original placeholder if we can't resolve it
	// This allows for literal string comparisons
	return originalPlaceholder, nil
}

// extractFunctionWrapper detects and extracts function wrappers (len, has)
func extractFunctionWrapper(placeholder string) (functionType, string) {
	if strings.HasPrefix(placeholder, LengthFunctionPrefix) && strings.HasSuffix(placeholder, FunctionSuffix) {
		inner := strings.TrimSuffix(strings.TrimPrefix(placeholder, LengthFunctionPrefix), FunctionSuffix)
		return functionLen, inner
	}
	if strings.HasPrefix(placeholder, HasFunctionPrefix) && strings.HasSuffix(placeholder, FunctionSuffix) {
		inner := strings.TrimSuffix(strings.TrimPrefix(placeholder, HasFunctionPrefix), FunctionSuffix)
		return functionHas, inner
	}
	return noFunction, placeholder
}

// resolveJSONPathPlaceholder handles [BODY].path and [BODY][index] placeholders
func resolveJSONPathPlaceholder(placeholder string, fn functionType, originalPlaceholder string, result *Result) (string, error) {
	// Extract the path after [BODY] (case insensitive)
	uppercasePlaceholder := strings.ToUpper(placeholder)
	path := ""
	if strings.HasPrefix(uppercasePlaceholder, BodyPlaceholder) {
		path = placeholder[len(BodyPlaceholder):]
	} else {
		path = strings.TrimPrefix(placeholder, BodyPlaceholder)
	}
	// Remove leading dot if present
	path = strings.TrimPrefix(path, ".")
	resolvedValue, resolvedLength, err := jsonpath.Eval(path, result.Body)
	if fn == functionHas {
		return strconv.FormatBool(err == nil), nil
	}
	if err != nil {
		return originalPlaceholder + " " + InvalidConditionElementSuffix, nil
	}
	if fn == functionLen {
		return strconv.Itoa(resolvedLength), nil
	}
	return resolvedValue, nil
}

// resolveContextPlaceholder handles [CONTEXT] placeholder resolution
func resolveContextPlaceholder(placeholder string, fn functionType, originalPlaceholder string, ctx *gontext.Gontext) (string, error) {
	contextPath := strings.TrimPrefix(placeholder, ContextPlaceholder)
	contextPath = strings.TrimPrefix(contextPath, ".")
	if contextPath == "" {
		if fn == functionHas {
			return "false", nil
		}
		return originalPlaceholder + " " + InvalidConditionElementSuffix, nil
	}
	value, err := ctx.Get(contextPath)
	if fn == functionHas {
		return strconv.FormatBool(err == nil), nil
	}
	if err != nil {
		return originalPlaceholder + " " + InvalidConditionElementSuffix, nil
	}
	if fn == functionLen {
		switch v := value.(type) {
		case string:
			return strconv.Itoa(len(v)), nil
		case []interface{}:
			return strconv.Itoa(len(v)), nil
		case map[string]interface{}:
			return strconv.Itoa(len(v)), nil
		default:
			return strconv.Itoa(len(fmt.Sprintf("%v", v))), nil
		}
	}
	return fmt.Sprintf("%v", value), nil
}

// formatWithFunction applies len/has functions to any value
func formatWithFunction(value string, fn functionType) string {
	switch fn {
	case functionHas:
		return strconv.FormatBool(value != "")
	case functionLen:
		return strconv.Itoa(len(value))
	default:
		return value
	}
}
