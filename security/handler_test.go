package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(200)
}

func TestHandlerWhenNotAuthenticated(t *testing.T) {
	handler := Handler(mockHandler, &Config{&BasicConfig{
		Username:           "john.doe",
		PasswordSha512Hash: "6b97ed68d14eb3f1aa959ce5d49c7dc612e1eb1dafd73b1e705847483fd6a6c809f2ceb4e8df6ff9984c6298ff0285cace6614bf8daa9f0070101b6c89899e22",
	}})
	request, _ := http.NewRequest("GET", "/api/v1/results", nil)
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusUnauthorized {
		t.Error("Expected code to be 401, but was", responseRecorder.Code)
	}
}

func TestHandlerWhenAuthenticated(t *testing.T) {
	handler := Handler(mockHandler, &Config{&BasicConfig{
		Username:           "john.doe",
		PasswordSha512Hash: "6b97ed68d14eb3f1aa959ce5d49c7dc612e1eb1dafd73b1e705847483fd6a6c809f2ceb4e8df6ff9984c6298ff0285cace6614bf8daa9f0070101b6c89899e22",
	}})
	request, _ := http.NewRequest("GET", "/api/v1/results", nil)
	request.SetBasicAuth("john.doe", "hunter2")
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Error("Expected code to be 200, but was", responseRecorder.Code)
	}
}

func TestHandlerWhenAuthenticatedWithBadCredentials(t *testing.T) {
	handler := Handler(mockHandler, &Config{&BasicConfig{
		Username:           "john.doe",
		PasswordSha512Hash: "6b97ed68d14eb3f1aa959ce5d49c7dc612e1eb1dafd73b1e705847483fd6a6c809f2ceb4e8df6ff9984c6298ff0285cace6614bf8daa9f0070101b6c89899e22",
	}})
	request, _ := http.NewRequest("GET", "/api/v1/results", nil)
	request.SetBasicAuth("john.doe", "bad-password")
	responseRecorder := httptest.NewRecorder()

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusUnauthorized {
		t.Error("Expected code to be 401, but was", responseRecorder.Code)
	}
}
