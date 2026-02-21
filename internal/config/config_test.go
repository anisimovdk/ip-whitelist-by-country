package config

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestConfigVersion(t *testing.T) {
	cfg := Config{}
	version := cfg.Version()

	if version == "" {
		t.Error("Version() returned empty string")
	}

	// Version should contain "Version:" prefix from GetFullVersion
	if !strings.Contains(version, "Version:") {
		t.Errorf("Version() should contain 'Version:', got: %s", version)
	}
}

func TestConfigDescription(t *testing.T) {
	cfg := Config{}
	desc := cfg.Description()

	expectedDesc := "IP Whitelist by Country - A service that provides IP network lists filtered by country"
	if desc != expectedDesc {
		t.Errorf("Description() = %q, want %q", desc, expectedDesc)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test that Config struct can be created with expected field types
	cfg := &Config{
		ServerPort:    "8080",
		AuthToken:     "",
		CacheDuration: "1h",
		ShowVersion:   false,
	}

	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "8080")
	}

	if cfg.AuthToken != "" {
		t.Errorf("AuthToken = %q, want empty string", cfg.AuthToken)
	}

	if cfg.CacheDuration != "1h" {
		t.Errorf("CacheDuration = %q, want %q", cfg.CacheDuration, "1h")
	}

	if cfg.ShowVersion != false {
		t.Errorf("ShowVersion = %v, want false", cfg.ShowVersion)
	}
}

func TestConfigWithCustomValues(t *testing.T) {
	cfg := &Config{
		ServerPort:    "9090",
		AuthToken:     "my-secret-token",
		CacheDuration: "24h",
		ShowVersion:   true,
	}

	if cfg.ServerPort != "9090" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "9090")
	}

	if cfg.AuthToken != "my-secret-token" {
		t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, "my-secret-token")
	}

	if cfg.CacheDuration != "24h" {
		t.Errorf("CacheDuration = %q, want %q", cfg.CacheDuration, "24h")
	}

	if cfg.ShowVersion != true {
		t.Errorf("ShowVersion = %v, want true", cfg.ShowVersion)
	}
}

func TestNewConfig_Defaults(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"app"}
	cfg := NewConfig()

	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "8080")
	}
	if cfg.AuthToken != "" {
		t.Errorf("AuthToken = %q, want empty string", cfg.AuthToken)
	}
	if cfg.CacheDuration != "1h" {
		t.Errorf("CacheDuration = %q, want %q", cfg.CacheDuration, "1h")
	}
	if cfg.ShowVersion {
		t.Errorf("ShowVersion = %v, want false", cfg.ShowVersion)
	}
}

func TestNewConfig_EnvOverrides(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"app"}
	t.Setenv("SERVER_PORT", "9091")
	t.Setenv("AUTH_TOKEN", "env-token")
	t.Setenv("CACHE_DURATION", "2h")

	cfg := NewConfig()
	if cfg.ServerPort != "9091" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "9091")
	}
	if cfg.AuthToken != "env-token" {
		t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, "env-token")
	}
	if cfg.CacheDuration != "2h" {
		t.Errorf("CacheDuration = %q, want %q", cfg.CacheDuration, "2h")
	}
}

func TestNewConfig_VersionFlagExits(t *testing.T) {
	origArgs := os.Args
	origExit := osExit
	origStdout := stdOut
	t.Cleanup(func() {
		os.Args = origArgs
		osExit = origExit
		stdOut = origStdout
	})

	var buf bytes.Buffer
	stdOut = &buf
	osExit = func(code int) { panic(code) }

	os.Args = []string{"app", "--version"}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected osExit to be called")
		}
		code, ok := r.(int)
		if !ok {
			t.Fatalf("expected panic with int exit code, got %T", r)
		}
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
		if !strings.Contains(buf.String(), "Version:") {
			t.Fatalf("expected version output to contain 'Version:', got %q", buf.String())
		}
	}()

	_ = NewConfig()
}
