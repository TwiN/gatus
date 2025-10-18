package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TwiN/gatus/v5/security"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// getUserSubjectFromRequest extracts the user subject from the authenticated request
// This works for both session-based and API key based authentication
func getUserSubjectFromRequest(c *fiber.Ctx, securityConfig *security.Config) (string, error) {
	if securityConfig == nil || securityConfig.OIDC == nil {
		return "", fmt.Errorf("OIDC not configured")
	}

	request, err := adaptor.ConvertRequest(c, false)
	if err != nil {
		return "", fmt.Errorf("failed to convert request: %w", err)
	}

	// Extract token (either session cookie or Bearer token)
	var token string
	authHeader := request.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	} else {
		sessionCookie, err := request.Cookie("gatus_session")
		if err == nil {
			token = sessionCookie.Value
		}
	}

	if token == "" {
		return "", fmt.Errorf("no authentication token found")
	}

	// If it's a session, get the subject from sessions
	if !strings.HasPrefix(token, security.APIKeyPrefix) {
		if subject, exists := security.GetSessionSubject(token); exists {
			return subject, nil
		}
		return "", fmt.Errorf("invalid session")
	}

	// If it's an API key, get the subject from the key itself
	allKeys, err := store.Get().GetAllAPIKeys()
	if err != nil {
		return "", fmt.Errorf("failed to get API keys: %w", err)
	}
	for _, apiKey := range allKeys {
		if security.ValidateAPIKey(apiKey, token) {
			return apiKey.UserSubject, nil
		}
	}

	return "", fmt.Errorf("invalid API key")
}

// ListAPIKeysHandler returns all API keys for the authenticated user
func ListAPIKeysHandler(securityConfig *security.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userSubject, err := getUserSubjectFromRequest(c, securityConfig)
		if err != nil {
			logr.Errorf("[api.ListAPIKeysHandler] Failed to get user subject: %v", err)
			return c.Status(401).SendString("Unauthorized")
		}

		apiKeys, err := store.Get().GetAPIKeys(userSubject)
		if err != nil {
			logr.Errorf("[api.ListAPIKeysHandler] Failed to get API keys for user %s: %v", userSubject, err)
			return c.Status(500).SendString("Failed to retrieve API keys")
		}

		c.Set("Content-Type", "application/json")
		responseBytes, err := json.Marshal(apiKeys)
		if err != nil {
			return c.Status(500).SendString(fmt.Sprintf(`{"error":"Failed to marshal response: %s"}`, err.Error()))
		}
		return c.Status(200).Send(responseBytes)
	}
}

// CreateAPIKeyHandler generates a new API key for the authenticated user
func CreateAPIKeyHandler(securityConfig *security.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userSubject, err := getUserSubjectFromRequest(c, securityConfig)
		if err != nil {
			logr.Errorf("[api.CreateAPIKeyHandler] Failed to get user subject: %v", err)
			return c.Status(401).SendString("Unauthorized")
		}

		// Parse request body for key name
		var requestBody struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&requestBody); err != nil {
			return c.Status(400).SendString("Invalid request body")
		}

		if requestBody.Name == "" {
			return c.Status(400).SendString("API key name is required")
		}

		// Generate the API key
		apiKeyWithToken, err := security.GenerateAPIKey(userSubject, requestBody.Name)
		if err != nil {
			logr.Errorf("[api.CreateAPIKeyHandler] Failed to generate API key: %v", err)
			return c.Status(500).SendString("Failed to generate API key")
		}

		// Store the API key (without the full token)
		if err := store.Get().CreateAPIKey(&apiKeyWithToken.APIKey); err != nil {
			logr.Errorf("[api.CreateAPIKeyHandler] Failed to store API key: %v", err)
			return c.Status(500).SendString("Failed to store API key")
		}

		logr.Infof("[api.CreateAPIKeyHandler] Created API key '%s' for user %s", requestBody.Name, userSubject)

		// Return the API key with the full token (only time it will be shown)
		c.Set("Content-Type", "application/json")
		responseBytes, err := json.Marshal(apiKeyWithToken)
		if err != nil {
			return c.Status(500).SendString(fmt.Sprintf(`{"error":"Failed to marshal response: %s"}`, err.Error()))
		}
		return c.Status(201).Send(responseBytes)
	}
}

// DeleteAPIKeyHandler deletes an API key
func DeleteAPIKeyHandler(securityConfig *security.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userSubject, err := getUserSubjectFromRequest(c, securityConfig)
		if err != nil {
			logr.Errorf("[api.DeleteAPIKeyHandler] Failed to get user subject: %v", err)
			return c.Status(401).SendString("Unauthorized")
		}

		keyID := c.Params("id")
		if keyID == "" {
			return c.Status(400).SendString("API key ID is required")
		}

		// Verify the key belongs to the user before deleting
		apiKeys, err := store.Get().GetAPIKeys(userSubject)
		if err != nil {
			logr.Errorf("[api.DeleteAPIKeyHandler] Failed to get API keys: %v", err)
			return c.Status(500).SendString("Failed to retrieve API keys")
		}

		keyBelongsToUser := false
		for _, key := range apiKeys {
			if key.ID == keyID {
				keyBelongsToUser = true
				break
			}
		}

		if !keyBelongsToUser {
			return c.Status(403).SendString("You don't have permission to delete this API key")
		}

		// Delete the key
		if err := store.Get().DeleteAPIKey(keyID); err != nil {
			logr.Errorf("[api.DeleteAPIKeyHandler] Failed to delete API key: %v", err)
			return c.Status(500).SendString("Failed to delete API key")
		}

		logr.Infof("[api.DeleteAPIKeyHandler] Deleted API key %s for user %s", keyID, userSubject)
		return c.Status(204).Send(nil)
	}
}
