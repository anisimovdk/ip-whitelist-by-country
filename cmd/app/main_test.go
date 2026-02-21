package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/anisimovdk/ip-whitelist-by-country/internal/config"
	"github.com/anisimovdk/ip-whitelist-by-country/internal/handler"
)

type mockProcessor struct {
	list []string
	err  error
}

func (m mockProcessor) GetIPListForCountry(countryCode string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.list, nil
}

func newTestServer(t *testing.T, cfg *config.Config) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	proc := mockProcessor{list: []string{"192.168.1.0/24"}}
	h := handler.NewHandler(proc, cfg)
	h.RegisterRoutesOn(mux)

	return httptest.NewServer(mux)
}

func TestIntegrationGetWithAuth(t *testing.T) {
	// Save original args and restore them after tests
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"app"}

	srv := newTestServer(t, &config.Config{AuthToken: "test-integration"})
	defer srv.Close()

	// Test with valid auth
	resp, err := http.Get(srv.URL + "/get?country=US&auth=test-integration")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test with invalid auth
	resp, err = http.Get(srv.URL + "/get?country=US&auth=wrong-token")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestIntegrationGetMissingCountry(t *testing.T) {
	// Save original args and restore them after tests
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"app"}

	srv := newTestServer(t, &config.Config{AuthToken: "test-integration"})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/get?auth=test-integration")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	// Drain body to ensure clean shutdown
	_, _ = io.ReadAll(resp.Body)
}
