package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRouter(t *testing.T) {
	router := CreateRouter(nil, nil, true)
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		Gzip         bool
	}
	scenarios := []Scenario{
		{
			Name:         "health",
			Path:         "/health",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "metrics",
			Path:         "/metrics",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "scripts",
			Path:         "/js/app.js",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "scripts-gzipped",
			Path:         "/js/app.js",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "index-redirect",
			Path:         "/index.html",
			ExpectedCode: http.StatusMovedPermanently,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request, _ := http.NewRequest("GET", scenario.Path, nil)
			if scenario.Gzip {
				request.Header.Set("Accept-Encoding", "gzip")
			}
			responseRecorder := httptest.NewRecorder()
			router.ServeHTTP(responseRecorder, request)
			if responseRecorder.Code != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, responseRecorder.Code)
			}
		})
	}
}
