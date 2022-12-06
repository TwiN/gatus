package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/config"
)

func TestCreateRouter(t *testing.T) {
	router := CreateRouter(&config.Config{Metrics: true})
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
			Name:         "favicon.ico",
			Path:         "/favicon.ico",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "app.js",
			Path:         "/js/app.js",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "app.js-gzipped",
			Path:         "/js/app.js",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "chunk-vendors.js",
			Path:         "/js/chunk-vendors.js",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "chunk-vendors.js-gzipped",
			Path:         "/js/chunk-vendors.js",
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
			request, _ := http.NewRequest("GET", scenario.Path, http.NoBody)
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
