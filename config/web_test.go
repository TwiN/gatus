package config

import "testing"

func TestWebConfig_SocketAddress(t *testing.T) {
	web := &webConfig{
		Address: "0.0.0.0",
		Port:    8081,
	}
	if web.SocketAddress() != "0.0.0.0:8081" {
		t.Errorf("expected %s, got %s", "0.0.0.0:8081", web.SocketAddress())
	}
}

func TestWebConfig_ContextRoot(t *testing.T) {
	const expected = "/status/"

	web := &webConfig{
		ContextRoot: "/status/",
	}

	web.validateAndSetDefaults()

	if web.CtxRoot() != expected {
		t.Errorf("expected %s, got %s", expected, web.CtxRoot())
	}
}

func TestWebConfig_ContextRootWithEscapableChars(t *testing.T) {
	const expected = "/s%3F=ta%20t%20u&s/"

	web := &webConfig{
		ContextRoot: "/s?=ta t u&s/",
	}

	web.validateAndSetDefaults()

	if web.CtxRoot() != expected {
		t.Errorf("expected %s, got %s", expected, web.CtxRoot())
	}
}

func TestWebConfig_ContextRootMultiPath(t *testing.T) {
	const expected = "/app/status"
	web := &webConfig{
		ContextRoot: "/app/status",
	}

	web.validateAndSetDefaults()

	if web.CtxRoot() != expected {
		t.Errorf("expected %s, got %s", expected, web.CtxRoot())
	}
}

func TestWebConfig_ContextRootAppendWithEmptyContextRoot(t *testing.T) {
	const expected = "/bla/"
	web := &webConfig{}

	web.validateAndSetDefaults()

	if web.AppendToCtxRoot("/bla/") != expected {
		t.Errorf("expected %s, got %s", expected, web.AppendToCtxRoot("/bla/"))
	}
}

func TestWebConfig_ContextRootAppendWithContext(t *testing.T) {
	const expected = "/app/status/bla/"
	web := &webConfig{
		ContextRoot: "/app/status",
	}

	web.validateAndSetDefaults()

	if web.AppendToCtxRoot("/bla/") != expected {
		t.Errorf("expected %s, got %s", expected, web.AppendToCtxRoot("/bla/"))
	}
}
