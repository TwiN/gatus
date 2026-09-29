package security

import (
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestValidateTokens(t *testing.T) {
	type Scenario struct {
		Name        string
		Tokens      []string
		ExpectValid bool
	}
	scenarios := []Scenario{
		{
			Name:        "nil",
			Tokens:      nil,
			ExpectValid: true,
		},
		{
			Name:        "valid",
			Tokens:      []string{"token1", "token2"},
			ExpectValid: true,
		},
		{
			Name:        "empty-token",
			Tokens:      []string{"token1", ""},
			ExpectValid: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if isValid := validateTokens(scenario.Tokens); isValid != scenario.ExpectValid {
				t.Errorf("expected %v, got %v", scenario.ExpectValid, isValid)
			}
		})
	}
}

func TestHasValidBearerToken(t *testing.T) {
	type Scenario struct {
		Name          string
		Tokens        []string
		Authorization string
		Expected      bool
	}
	scenarios := []Scenario{
		{
			Name:          "no-tokens",
			Tokens:        nil,
			Authorization: "Bearer token1",
			Expected:      false,
		},
		{
			Name:          "no-header",
			Tokens:        []string{"token1"},
			Authorization: "",
			Expected:      false,
		},
		{
			Name:          "empty-bearer-token",
			Tokens:        []string{""},
			Authorization: "Bearer ",
			Expected:      false,
		},
		{
			Name:          "matching-token",
			Tokens:        []string{"token1", "token2"},
			Authorization: "Bearer token2",
			Expected:      true,
		},
		{
			Name:          "matching-token-with-surrounding-whitespace",
			Tokens:        []string{"token1"},
			Authorization: "Bearer  token1 ",
			Expected:      true,
		},
		{
			Name:          "non-matching-token",
			Tokens:        []string{"token1"},
			Authorization: "Bearer token2",
			Expected:      false,
		},
		{
			Name:          "partial-token",
			Tokens:        []string{"my-secret-token"},
			Authorization: "Bearer my-secret",
			Expected:      false,
		},
		{
			Name:          "lowercase-bearer",
			Tokens:        []string{"token1"},
			Authorization: "bearer token1",
			Expected:      false,
		},
		{
			Name:          "basic-auth",
			Tokens:        []string{"token1"},
			Authorization: "Basic dG9rZW4xOg==",
			Expected:      false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(ctx *fiber.Ctx) error {
				if hasValidBearerToken(ctx, scenario.Tokens) {
					return ctx.SendStatus(200)
				}
				return ctx.SendStatus(401)
			})
			if hasValidBearerToken := testRequest(t, app, scenario.Authorization, false) == 200; hasValidBearerToken != scenario.Expected {
				t.Errorf("expected %v, got %v", scenario.Expected, hasValidBearerToken)
			}
		})
	}
}
