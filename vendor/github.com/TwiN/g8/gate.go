package g8

import (
	"context"
	"net/http"
	"strings"
)

const (
	// AuthorizationHeader is the header in which g8 looks for the authorization bearer token
	AuthorizationHeader = "Authorization"

	// DefaultUnauthorizedResponseBody is the default response body returned if a request was sent with a missing or invalid token
	DefaultUnauthorizedResponseBody = "token is missing or invalid"

	// DefaultTooManyRequestsResponseBody is the default response body returned if a request exceeded the allowed rate limit
	DefaultTooManyRequestsResponseBody = "too many requests"

	// TokenContextKey is the key used to store the token in the context.
	TokenContextKey = "g8.token"
)

// Gate is lock to the front door of your API, letting only those you allow through.
type Gate struct {
	authorizationService     *AuthorizationService
	unauthorizedResponseBody []byte

	customTokenExtractorFunc func(request *http.Request) string

	rateLimiter                 *RateLimiter
	tooManyRequestsResponseBody []byte
}

// Deprecated: use New instead.
func NewGate(authorizationService *AuthorizationService) *Gate {
	return &Gate{
		authorizationService:        authorizationService,
		unauthorizedResponseBody:    []byte(DefaultUnauthorizedResponseBody),
		tooManyRequestsResponseBody: []byte(DefaultTooManyRequestsResponseBody),
	}
}

// New creates a new Gate.
func New() *Gate {
	return &Gate{
		unauthorizedResponseBody:    []byte(DefaultUnauthorizedResponseBody),
		tooManyRequestsResponseBody: []byte(DefaultTooManyRequestsResponseBody),
	}
}

// WithAuthorizationService sets the authorization service to use.
//
// If there is no authorization service, Gate will not enforce authorization.
func (gate *Gate) WithAuthorizationService(authorizationService *AuthorizationService) *Gate {
	gate.authorizationService = authorizationService
	return gate
}

// WithCustomUnauthorizedResponseBody sets a custom response body when Gate determines that a request must be blocked
func (gate *Gate) WithCustomUnauthorizedResponseBody(unauthorizedResponseBody []byte) *Gate {
	gate.unauthorizedResponseBody = unauthorizedResponseBody
	return gate
}

// WithCustomTokenExtractor allows the specification of a custom function to extract a token from a request.
// If a custom token extractor is not specified, the token will be extracted from the Authorization header.
//
// For instance, if you're using a session cookie, you can extract the token from the cookie like so:
//     	authorizationService := g8.NewAuthorizationService()
//     	customTokenExtractorFunc := func(request *http.Request) string {
//     		sessionCookie, err := request.Cookie("session")
//     		if err != nil {
//     			return ""
//     		}
//     		return sessionCookie.Value
//     	}
//     	gate := g8.New().WithAuthorizationService(authorizationService).WithCustomTokenExtractor(customTokenExtractorFunc)
//
// You would normally use this with a client provider that matches whatever need you have.
// For example, if you're using a session cookie, your client provider would retrieve the user from the session ID
// extracted by this custom token extractor.
//
// Note that for the sake of convenience, the token extracted from the request is passed the protected handlers request
// context under the key TokenContextKey. This is especially useful if the token is in fact a session ID.
func (gate *Gate) WithCustomTokenExtractor(customTokenExtractorFunc func(request *http.Request) string) *Gate {
	gate.customTokenExtractorFunc = customTokenExtractorFunc
	return gate
}

// WithRateLimit adds rate limiting to the Gate
//
// If you just want to use a gate for rate limiting purposes:
//    gate := g8.New().WithRateLimit(50)
//
func (gate *Gate) WithRateLimit(maximumRequestsPerSecond int) *Gate {
	gate.rateLimiter = NewRateLimiter(maximumRequestsPerSecond)
	return gate
}

// Protect secures a handler, requiring requests going through to have a valid Authorization Bearer token.
// Unlike ProtectWithPermissions, Protect will allow access to any registered tokens, regardless of their permissions
// or lack thereof.
//
// Example:
//    gate := g8.New().WithAuthorizationService(g8.NewAuthorizationService().WithToken("token"))
//    router := http.NewServeMux()
//    // Without protection
//    router.Handle("/handle", yourHandler)
//    // With protection
//    router.Handle("/handle", gate.Protect(yourHandler))
//
// The token extracted from the request is passed to the handlerFunc request context under the key TokenContextKey
func (gate *Gate) Protect(handler http.Handler) http.Handler {
	return gate.ProtectWithPermissions(handler, nil)
}

// ProtectWithPermissions secures a handler, requiring requests going through to have a valid Authorization Bearer token
// as well as a slice of permissions that must be met.
//
// Example:
//    gate := g8.New().WithAuthorizationService(g8.NewAuthorizationService().WithClient(g8.NewClient("token").WithPermission("admin")))
//    router := http.NewServeMux()
//    // Without protection
//    router.Handle("/handle", yourHandler)
//    // With protection
//    router.Handle("/handle", gate.ProtectWithPermissions(yourHandler, []string{"admin"}))
//
// The token extracted from the request is passed to the handlerFunc request context under the key TokenContextKey
func (gate *Gate) ProtectWithPermissions(handler http.Handler, permissions []string) http.Handler {
	return gate.ProtectFuncWithPermissions(func(writer http.ResponseWriter, request *http.Request) {
		handler.ServeHTTP(writer, request)
	}, permissions)
}

// ProtectWithPermission does the same thing as ProtectWithPermissions, but for a single permission instead of a
// slice of permissions
//
// See ProtectWithPermissions for further documentation
func (gate *Gate) ProtectWithPermission(handler http.Handler, permission string) http.Handler {
	return gate.ProtectFuncWithPermissions(func(writer http.ResponseWriter, request *http.Request) {
		handler.ServeHTTP(writer, request)
	}, []string{permission})
}

// ProtectFunc secures a handlerFunc, requiring requests going through to have a valid Authorization Bearer token.
// Unlike ProtectFuncWithPermissions, ProtectFunc will allow access to any registered tokens, regardless of their
// permissions or lack thereof.
//
// Example:
//    gate := g8.New().WithAuthorizationService(g8.NewAuthorizationService().WithToken("token"))
//    router := http.NewServeMux()
//    // Without protection
//    router.HandleFunc("/handle", yourHandlerFunc)
//    // With protection
//    router.HandleFunc("/handle", gate.ProtectFunc(yourHandlerFunc))
//
// The token extracted from the request is passed to the handlerFunc request context under the key TokenContextKey
func (gate *Gate) ProtectFunc(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return gate.ProtectFuncWithPermissions(handlerFunc, nil)
}

// ProtectFuncWithPermissions secures a handler, requiring requests going through to have a valid Authorization Bearer
// token as well as a slice of permissions that must be met.
//
// Example:
//    gate := g8.New().WithAuthorizationService(g8.NewAuthorizationService().WithClient(g8.NewClient("token").WithPermission("admin")))
//    router := http.NewServeMux()
//    // Without protection
//    router.HandleFunc("/handle", yourHandlerFunc)
//    // With protection
//    router.HandleFunc("/handle", gate.ProtectFuncWithPermissions(yourHandlerFunc, []string{"admin"}))
//
// The token extracted from the request is passed to the handlerFunc request context under the key TokenContextKey
func (gate *Gate) ProtectFuncWithPermissions(handlerFunc http.HandlerFunc, permissions []string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if gate.rateLimiter != nil {
			if !gate.rateLimiter.Try() {
				writer.WriteHeader(http.StatusTooManyRequests)
				_, _ = writer.Write(gate.tooManyRequestsResponseBody)
				return
			}
		}
		if gate.authorizationService != nil {
			var token string
			if gate.customTokenExtractorFunc != nil {
				token = gate.customTokenExtractorFunc(request)
			} else {
				token = extractTokenFromRequest(request)
			}
			if !gate.authorizationService.IsAuthorized(token, permissions) {
				writer.WriteHeader(http.StatusUnauthorized)
				_, _ = writer.Write(gate.unauthorizedResponseBody)
				return
			}
			request = request.WithContext(context.WithValue(request.Context(), TokenContextKey, token))
		}
		handlerFunc(writer, request)
	}
}

// ProtectFuncWithPermission does the same thing as ProtectFuncWithPermissions, but for a single permission instead of a
// slice of permissions
//
// See ProtectFuncWithPermissions for further documentation
func (gate *Gate) ProtectFuncWithPermission(handlerFunc http.HandlerFunc, permission string) http.HandlerFunc {
	return gate.ProtectFuncWithPermissions(handlerFunc, []string{permission})
}

// extractTokenFromRequest extracts the bearer token from the AuthorizationHeader
func extractTokenFromRequest(request *http.Request) string {
	return strings.TrimPrefix(request.Header.Get(AuthorizationHeader), "Bearer ")
}
