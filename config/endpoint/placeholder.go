package endpoint

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	// HeaderPlaceholder is a placeholder for the value of an HTTP response header.
	// Header names are case-insensitive.
	// Usage: [HEADER].Content-Type
	HeaderPlaceholder = "[HEADER]"

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

	// AgeFunctionPrefix is the prefix for the age function, which returns the number of milliseconds elapsed since
	// the timestamp resolved from the wrapped placeholder. By default, supported timestamp formats are HTTP dates
	// (RFC 1123, RFC 850, ANSI C), RFC 3339 and Unix epochs in seconds or milliseconds. A Go time layout may be passed
	// as second argument to parse any other format; timestamps without a timezone are interpreted as UTC.
	//
	// Usage: age([HEADER].Last-Modified) < 10m, age([BODY].updated_at, 2006-01-02 15:04:05) < 1h
	AgeFunctionPrefix = "age("

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
	// Note that not all functions are handled here. Only len(), has() and age() directly impact the handler
	// e.g. "len([BODY].name) > 0" vs pat() or any(), which would be used like "[BODY].name == pat(john*)"

	noFunction functionType = iota
	functionLen
	functionHas
	functionAge
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
//   - [HEADER].name: HTTP response header value (e.g., [HEADER].Content-Type)
//   - [CONTEXT].path: Suite context values (e.g., [CONTEXT].user_id, [CONTEXT].session_token)
//
// Function wrappers:
//   - len(placeholder): Returns the length of the resolved value
//   - has(placeholder): Returns "true" if the placeholder exists and is non-empty, "false" otherwise
//   - age(placeholder[, layout]): Parses the placeholder's value as a timestamp and returns the milliseconds elapsed since then
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
	if fn == functionAge {
		return resolveAgeFunction(placeholder, originalPlaceholder, result, ctx)
	}

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

	// Handle HEADER placeholders
	if strings.HasPrefix(uppercasePlaceholder, HeaderPlaceholder+".") {
		return resolveHeaderPlaceholder(placeholder, fn, originalPlaceholder, result), nil
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

// extractFunctionWrapper detects and extracts function wrappers (len, has, age)
func extractFunctionWrapper(placeholder string) (functionType, string) {
	if strings.HasPrefix(placeholder, LengthFunctionPrefix) && strings.HasSuffix(placeholder, FunctionSuffix) {
		inner := strings.TrimSuffix(strings.TrimPrefix(placeholder, LengthFunctionPrefix), FunctionSuffix)
		return functionLen, inner
	}
	if strings.HasPrefix(placeholder, HasFunctionPrefix) && strings.HasSuffix(placeholder, FunctionSuffix) {
		inner := strings.TrimSuffix(strings.TrimPrefix(placeholder, HasFunctionPrefix), FunctionSuffix)
		return functionHas, inner
	}
	if strings.HasPrefix(placeholder, AgeFunctionPrefix) && strings.HasSuffix(placeholder, FunctionSuffix) {
		inner := strings.TrimSuffix(strings.TrimPrefix(placeholder, AgeFunctionPrefix), FunctionSuffix)
		return functionAge, inner
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

// resolveHeaderPlaceholder handles [HEADER].name placeholders
//
// If the header has multiple values, only the first one is used, like http.Header.Get
func resolveHeaderPlaceholder(placeholder string, fn functionType, originalPlaceholder string, result *Result) string {
	name := placeholder[len(HeaderPlaceholder)+1:]
	values := result.Headers.Values(name)
	if fn == functionHas {
		return strconv.FormatBool(len(values) > 0)
	}
	if len(values) == 0 {
		return originalPlaceholder + " " + InvalidConditionElementSuffix
	}
	return formatWithFunction(values[0], fn)
}

// resolveAgeFunction handles age(placeholder) by resolving the wrapped placeholder to a timestamp and returning the
// number of milliseconds elapsed since then
func resolveAgeFunction(arguments, originalPlaceholder string, result *Result, ctx *gontext.Gontext) (string, error) {
	placeholder, layout, _ := strings.Cut(arguments, ",")
	// Other placeholders don't resolve into timestamps (e.g. [CERTIFICATE_EXPIRATION] is a duration), so the result
	// would be meaningless
	uppercasePlaceholder := strings.ToUpper(strings.TrimSpace(placeholder))
	if !strings.HasPrefix(uppercasePlaceholder, BodyPlaceholder) && !strings.HasPrefix(uppercasePlaceholder, HeaderPlaceholder+".") && !strings.HasPrefix(uppercasePlaceholder, ContextPlaceholder+".") {
		return originalPlaceholder + " " + InvalidConditionElementSuffix, fmt.Errorf("%s: %s only supports the %s, %s and %s placeholders", originalPlaceholder, strings.TrimSuffix(AgeFunctionPrefix, "("), BodyPlaceholder, HeaderPlaceholder, ContextPlaceholder)
	}
	value, err := ResolvePlaceholder(placeholder, result, ctx)
	if err != nil || strings.HasSuffix(value, InvalidConditionElementSuffix) {
		return originalPlaceholder + " " + InvalidConditionElementSuffix, err
	}
	timestamp, ok := parseTimestamp(value, strings.TrimSpace(layout))
	if !ok {
		return originalPlaceholder + " " + InvalidConditionElementSuffix, nil
	}
	return strconv.FormatInt(time.Since(timestamp).Milliseconds(), 10), nil
}

// parseTimestamp parses a timestamp using the given Go time layout or, if no layout is provided, as an HTTP date, an
// RFC 3339 timestamp or a Unix epoch in seconds or milliseconds
func parseTimestamp(value, layout string) (time.Time, bool) {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if layout != "" {
		t, err := time.Parse(layout, value)
		return t, err == nil
	}
	if t, err := http.ParseTime(value); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, true
	}
	// JSON numbers may be rendered in scientific notation (e.g. 1.7274e+09), so they're parsed as floats
	if epoch, err := strconv.ParseFloat(value, 64); err == nil && epoch > 0 && !math.IsInf(epoch, 0) {
		// Values below 1e12 are assumed to be in seconds, and anything else in milliseconds: 1e12 seconds is around the
		// year 33658, whereas 1e12 milliseconds is around the year 2001
		if epoch >= 1e12 {
			return time.UnixMilli(int64(epoch)), true
		}
		return time.UnixMicro(int64(epoch * 1e6)), true
	}
	return time.Time{}, false
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
