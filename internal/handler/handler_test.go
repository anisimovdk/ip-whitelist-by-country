package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anisimovdk/ip-whitelist-by-country/internal/config"
)

// MockProcessor is a mock implementation of the processor interface for testing
type MockProcessor struct {
	ipLists map[string][]string
	err     error
}

// GetIPListForCountry is a mock implementation that returns test data
func (m *MockProcessor) GetIPListForCountry(countryCode string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.ipLists[countryCode], nil
}

func TestNewHandler(t *testing.T) {
	mockProc := &MockProcessor{}
	cfg := &config.Config{
		ServerPort: "8080",
		AuthToken:  "test-token",
	}

	h := NewHandler(mockProc, cfg)

	if h == nil {
		t.Fatal("NewHandler returned nil")
	}

	if h.processor == nil {
		t.Error("Handler processor is nil")
	}

	if h.config == nil {
		t.Error("Handler config is nil")
	}

	if h.config.AuthToken != "test-token" {
		t.Errorf("Handler config AuthToken = %q, want %q", h.config.AuthToken, "test-token")
	}
}

func TestRegisterRoutes(t *testing.T) {
	oldMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = oldMux })

	mockIPList := []string{"192.168.1.0/24"}
	mockProc := &MockProcessor{ipLists: map[string][]string{"US": mockIPList}}
	cfg := &config.Config{ServerPort: "8080", AuthToken: ""}

	h := NewHandler(mockProc, cfg)
	h.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/get?country=US", nil)
	rr := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "192.168.1.0/24\n" {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestGetIpListHandlerWithoutAuth(t *testing.T) {
	// Create mock data
	mockIPList := []string{"192.168.1.0/24", "10.0.0.0/8"}

	// Create a mock processor
	mockProc := &MockProcessor{
		ipLists: map[string][]string{
			"US": mockIPList,
		},
	}

	// Create a config with no auth token
	cfg := &config.Config{
		ServerPort: "8080",
		AuthToken:  "", // No auth required
	}

	// Create the handler
	h := NewHandler(mockProc, cfg)

	// Create a request
	req, err := http.NewRequest("GET", "/get?country=US", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(h.getIpListHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body (should contain the IP list)
	expected := "192.168.1.0/24\n10.0.0.0/8\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestGetIpListHandlerWithAuth(t *testing.T) {
	// Create mock data
	mockIPList := []string{"192.168.1.0/24", "10.0.0.0/8"}

	// Create a mock processor
	mockProc := &MockProcessor{
		ipLists: map[string][]string{
			"US": mockIPList,
		},
	}

	// Create a config with auth token
	cfg := &config.Config{
		ServerPort: "8080",
		AuthToken:  "test-token", // Auth required
	}

	// Create the handler
	h := NewHandler(mockProc, cfg)

	// Test cases
	testCases := []struct {
		name           string
		authToken      string
		expectedStatus int
	}{
		{
			name:           "Valid auth token",
			authToken:      "test-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid auth token",
			authToken:      "wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing auth token",
			authToken:      "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a request
			reqURL := "/get?country=US"
			if tc.authToken != "" {
				reqURL += "&auth=" + tc.authToken
			}

			req, err := http.NewRequest("GET", reqURL, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler
			handler := http.HandlerFunc(h.getIpListHandler)
			handler.ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			// If we expect success, check the response body
			if tc.expectedStatus == http.StatusOK {
				expected := "192.168.1.0/24\n10.0.0.0/8\n"
				if rr.Body.String() != expected {
					t.Errorf("handler returned unexpected body: got %v want %v",
						rr.Body.String(), expected)
				}
			}
		})
	}
}

func TestGetIpListHandlerMissingCountry(t *testing.T) {
	// Create a mock processor
	mockProc := &MockProcessor{}

	// Create a config
	cfg := &config.Config{
		ServerPort: "8080",
		AuthToken:  "",
	}

	// Create the handler
	h := NewHandler(mockProc, cfg)

	// Create a request without country parameter
	req, err := http.NewRequest("GET", "/get", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(h.getIpListHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func TestGetIpListHandlerWrongMethod(t *testing.T) {
	// Create a mock processor
	mockProc := &MockProcessor{}

	// Create a config
	cfg := &config.Config{
		ServerPort: "8080",
		AuthToken:  "",
	}

	// Create the handler
	h := NewHandler(mockProc, cfg)

	// Create a POST request
	req, err := http.NewRequest("POST", "/get?country=US", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(h.getIpListHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusMethodNotAllowed)
	}
}

func TestGetIpListHandlerProcessorError(t *testing.T) {
	// Create a mock processor that returns an error
	mockProc := &MockProcessor{
		err: errors.New("database connection failed"),
	}

	// Create a config with no auth token
	cfg := &config.Config{
		ServerPort: "8080",
		AuthToken:  "", // No auth required
	}

	// Create the handler
	h := NewHandler(mockProc, cfg)

	// Create a request
	req, err := http.NewRequest("GET", "/get?country=US", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(h.getIpListHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code - should be internal server error
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	// Check that error message is in response
	if rr.Body.String() == "" {
		t.Error("Expected error message in response body")
	}
}

func TestGetIpListHandlerEmptyResult(t *testing.T) {
	// Create a mock processor that returns empty result for unknown country
	mockProc := &MockProcessor{
		ipLists: map[string][]string{
			"US": {"192.168.1.0/24"},
		},
	}

	// Create a config with no auth token
	cfg := &config.Config{
		ServerPort: "8080",
		AuthToken:  "", // No auth required
	}

	// Create the handler
	h := NewHandler(mockProc, cfg)

	// Create a request for a country not in the mock data
	req, err := http.NewRequest("GET", "/get?country=XX", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(h.getIpListHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code - should be OK even for empty result
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Response body should be empty (no IPs for this country)
	if rr.Body.String() != "" {
		t.Errorf("Expected empty body for unknown country, got %q", rr.Body.String())
	}
}

func TestGetIpListHandlerContentType(t *testing.T) {
	// Create mock data
	mockIPList := []string{"192.168.1.0/24"}

	// Create a mock processor
	mockProc := &MockProcessor{
		ipLists: map[string][]string{
			"US": mockIPList,
		},
	}

	// Create a config with no auth token
	cfg := &config.Config{
		ServerPort: "8080",
		AuthToken:  "",
	}

	// Create the handler
	h := NewHandler(mockProc, cfg)

	// Create a request
	req, err := http.NewRequest("GET", "/get?country=US", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(h.getIpListHandler)
	handler.ServeHTTP(rr, req)

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("handler returned wrong content type: got %v want %v",
			contentType, "text/plain")
	}
}
