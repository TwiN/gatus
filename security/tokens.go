package security

import (
	"crypto/subtle"
	"slices"
	"strings"
	"unicode"

	"github.com/gofiber/fiber/v2"
)

// validateTokens returns whether none of the tokens are empty or contain whitespace
func validateTokens(tokens []string) bool {
	return !slices.ContainsFunc(tokens, func(token string) bool {
		return len(token) == 0 || strings.ContainsFunc(token, unicode.IsSpace)
	})
}

// hasValidBearerToken returns whether the request has a bearer token matching one of the tokens passed
func hasValidBearerToken(ctx *fiber.Ctx, tokens []string) bool {
	token, ok := strings.CutPrefix(ctx.Get("Authorization"), "Bearer ")
	if !ok {
		return false
	}
	token = strings.TrimSpace(token)
	if len(token) == 0 {
		return false
	}
	for _, validToken := range tokens {
		if subtle.ConstantTimeCompare([]byte(validToken), []byte(token)) == 1 {
			return true
		}
	}
	return false
}
