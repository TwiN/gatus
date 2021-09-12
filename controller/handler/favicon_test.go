package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFavIcon(t *testing.T) {
	router := CreateRouter("../../web/static", nil, nil, false)
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
	}
	scenarios := []Scenario{
		{
			Name:         "favicon",
			Path:         "/favicon.ico",
			ExpectedCode: http.StatusOK,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request, _ := http.NewRequest("GET", scenario.Path, nil)
			responseRecorder := httptest.NewRecorder()
			router.ServeHTTP(responseRecorder, request)
			if responseRecorder.Code != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, responseRecorder.Code)
			}
		})
	}
}
