package controller

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/config/web"
	"github.com/TwinProduction/gatus/core"
)

func TestHandle(t *testing.T) {
	cfg := &config.Config{
		Web: &web.Config{
			Address: "0.0.0.0",
			Port:    rand.Intn(65534),
		},
		Services: []*core.Service{
			{
				Name:  "frontend",
				Group: "core",
			},
			{
				Name:  "backend",
				Group: "core",
			},
		},
	}
	_ = os.Setenv("ROUTER_TEST", "true")
	_ = os.Setenv("ENVIRONMENT", "dev")
	defer os.Clearenv()
	Handle(cfg.Security, cfg.Web, cfg.UI, cfg.Metrics)
	defer Shutdown()
	request, _ := http.NewRequest("GET", "/health", nil)
	responseRecorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusOK {
		t.Error("expected GET /health to return status code 200")
	}
	if server == nil {
		t.Fatal("server should've been set (but because we set ROUTER_TEST, it shouldn't have been started)")
	}
}

func TestShutdown(t *testing.T) {
	// Pretend that we called controller.Handle(), which initializes the server variable
	server = &http.Server{}
	Shutdown()
	if server != nil {
		t.Error("server should've been shut down")
	}
}
