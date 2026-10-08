package security

import (
	"crypto/subtle"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// validateTokens returns whether none of the tokens are empty
func validateTokens(tokens []string) bool {
	return !slices.Contains(tokens, "")
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
