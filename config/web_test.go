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

func TestWebConfig_ContextRootEmpty(t *testing.T) {
	const expected = "/"

	web := &webConfig{
		ContextRoot: "",
	}

	web.validateAndSetDefaults()

	if web.ContextRoot != expected {
		t.Errorf("expected %s, got %s", expected, web.ContextRoot)
	}
}

func TestWebConfig_ContextRoot(t *testing.T) {
	const expected = "/status/"

	web := &webConfig{
		ContextRoot: "/status/",
	}

	web.validateAndSetDefaults()

	if web.ContextRoot != expected {
		t.Errorf("expected %s, got %s", expected, web.ContextRoot)
	}
}

func TestWebConfig_ContextRootInvalid(t *testing.T) {
	defer func() { recover() }()

	web := &webConfig{
		ContextRoot: "/s?=ta t u&s/",
	}

	web.validateAndSetDefaults()

	t.Fatal("Should've panicked because the configuration specifies an invalid context root")
}

func TestWebConfig_ContextRootMultiPath(t *testing.T) {
	const expected = "/app/status/"
	web := &webConfig{
		ContextRoot: "/app/status",
	}

	web.validateAndSetDefaults()

	if web.ContextRoot != expected {
		t.Errorf("expected %s, got %s", expected, web.ContextRoot)
	}
}

func TestWebConfig_ContextRootAppendWithEmptyContextRoot(t *testing.T) {
	const expected = "/bla/"
	web := &webConfig{}

	web.validateAndSetDefaults()

	if web.AppendToContexRoot("/bla/") != expected {
		t.Errorf("expected %s, got %s", expected, web.AppendToContexRoot("/bla/"))
	}
}

func TestWebConfig_ContextRootAppendWithContext(t *testing.T) {
	const expected = "/app/status/bla/"
	web := &webConfig{
		ContextRoot: "/app/status",
	}

	web.validateAndSetDefaults()

	if web.AppendToContexRoot("/bla/") != expected {
		t.Errorf("expected %s, got %s", expected, web.AppendToContexRoot("/bla/"))
	}
}
