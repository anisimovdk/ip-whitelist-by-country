package ipdata

import (
	"bytes"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/anisimovdk/ip-whitelist-by-country/internal/config"
)

// MockHTTPClient is a mock implementation of HTTPClient for testing
type MockHTTPClient struct {
	ShouldError  bool
	ErrorMsg     string
	ResponseBody string
	StatusCode   int
	Body         io.ReadCloser
	CallCount    int
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.CallCount++
	if m.ShouldError {
		return nil, errors.New(m.ErrorMsg)
	}

	statusCode := m.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	body := m.Body
	if body == nil {
		body = io.NopCloser(bytes.NewBufferString(m.ResponseBody))
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       body,
	}, nil
}

// Create a test processor for testing
func createTestProcessor() *Processor {
	cfg := &config.Config{
		ServerPort:    "8080",
		AuthToken:     "test-token",
		CacheDuration: "1h",
	}

	return &Processor{
		cache:      make(map[string][]string),
		cacheTime:  time.Time{},
		config:     cfg,
		cacheTTL:   1 * time.Hour,
		httpClient: &MockHTTPClient{ShouldError: true, ErrorMsg: "mock download error"},
	}
}

// createTestProcessorWithMockData creates a processor with a successful HTTP mock
func createTestProcessorWithMockData(responseBody string) *Processor {
	cfg := &config.Config{
		ServerPort:    "8080",
		AuthToken:     "test-token",
		CacheDuration: "1h",
	}

	return &Processor{
		cache:     make(map[string][]string),
		cacheTime: time.Time{},
		config:    cfg,
		cacheTTL:  1 * time.Hour,
		httpClient: &MockHTTPClient{
			ShouldError:  false,
			ResponseBody: responseBody,
			StatusCode:   http.StatusOK,
		},
	}
}

func TestNewProcessor_InvalidCacheDurationFallsBack(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"app"}
	t.Setenv("CACHE_DURATION", "not-a-duration")

	processor := NewProcessor()
	if processor == nil {
		t.Fatal("NewProcessor returned nil")
	}
	if processor.cache == nil {
		t.Fatal("processor cache is nil")
	}
	if processor.httpClient == nil {
		t.Fatal("processor httpClient is nil")
	}
	if processor.cacheTTL != 1*time.Hour {
		t.Fatalf("cacheTTL = %v, want %v", processor.cacheTTL, 1*time.Hour)
	}
}

func TestNewProcessorWithClient_UsesClientAndTTL(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"app"}
	t.Setenv("CACHE_DURATION", "2h")

	mockClient := &MockHTTPClient{ResponseBody: ""}
	processor := NewProcessorWithClient(mockClient)
	if processor.httpClient != mockClient {
		t.Fatal("expected custom http client to be used")
	}
	if processor.cacheTTL != 2*time.Hour {
		t.Fatalf("cacheTTL = %v, want %v", processor.cacheTTL, 2*time.Hour)
	}
}

func TestNewProcessorWithClient_InvalidCacheDurationFallsBack(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"app"}
	t.Setenv("CACHE_DURATION", "not-a-duration")

	mockClient := &MockHTTPClient{ResponseBody: ""}
	processor := NewProcessorWithClient(mockClient)
	if processor.cacheTTL != 1*time.Hour {
		t.Fatalf("cacheTTL = %v, want %v", processor.cacheTTL, 1*time.Hour)
	}
}

type errReadCloser struct{}

func (errReadCloser) Read(p []byte) (int, error) { return 0, errors.New("read error") }
func (errReadCloser) Close() error               { return nil }

func TestDownloadAndProcessData_CacheShortCircuit(t *testing.T) {
	processor := &Processor{
		cache:      map[string][]string{"US": {"1.1.1.0/24"}},
		cacheTime:  time.Now(),
		cacheTTL:   1 * time.Hour,
		httpClient: &MockHTTPClient{ResponseBody: "should not be used"},
		config:     &config.Config{CacheDuration: "1h"},
	}

	err := processor.downloadAndProcessData()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	mc := processor.httpClient.(*MockHTTPClient)
	if mc.CallCount != 0 {
		t.Fatalf("expected http client not to be called, CallCount=%d", mc.CallCount)
	}
}

func TestDownloadAndProcessData_RequestCreateError(t *testing.T) {
	oldURL := ripeURL
	t.Cleanup(func() { ripeURL = oldURL })
	ripeURL = "http://[::1" // invalid URL

	processor := &Processor{
		cache:      make(map[string][]string),
		cacheTime:  time.Time{},
		cacheTTL:   0,
		httpClient: &MockHTTPClient{ResponseBody: ""},
		config:     &config.Config{CacheDuration: "1h"},
	}

	err := processor.downloadAndProcessData()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadAndProcessData_HTTPClientError(t *testing.T) {
	processor := &Processor{
		cache:     make(map[string][]string),
		cacheTime: time.Time{},
		cacheTTL:  0,
		httpClient: &MockHTTPClient{
			ShouldError: true,
			ErrorMsg:    "boom",
		},
		config: &config.Config{CacheDuration: "1h"},
	}

	err := processor.downloadAndProcessData()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to download data") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadAndProcessData_Non200Response(t *testing.T) {
	processor := &Processor{
		cache:     make(map[string][]string),
		cacheTime: time.Time{},
		cacheTTL:  0,
		httpClient: &MockHTTPClient{
			StatusCode:   http.StatusInternalServerError,
			ResponseBody: "",
		},
		config: &config.Config{CacheDuration: "1h"},
	}

	err := processor.downloadAndProcessData()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "received non-200 response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadAndProcessData_ScannerError(t *testing.T) {
	processor := &Processor{
		cache:     make(map[string][]string),
		cacheTime: time.Time{},
		cacheTTL:  0,
		httpClient: &MockHTTPClient{
			StatusCode: http.StatusOK,
			Body:       errReadCloser{},
		},
		config: &config.Config{CacheDuration: "1h"},
	}

	err := processor.downloadAndProcessData()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "error reading response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadAndProcessData_SuccessParsesAndCaches(t *testing.T) {
	data := strings.Join([]string{
		"# comment",
		"",
		"invalid|line",
		"ripencc|US|ipv4|192.168.0.0|256|20220101|allocated",
		"ripencc|FR|ipv6|2001:db8::|1|20220101|allocated",
		"apnic|CN|ipv4|172.16.0.0|4096|20220101|allocated",
		"ripencc|DE|ipv4|10.0.0.0|notanint|20220101|allocated",
		"ripencc|DE|ipv4|10.0.0.0|65536|20220101|allocated",
	}, "\n")

	processor := &Processor{
		cache:     make(map[string][]string),
		cacheTime: time.Time{},
		cacheTTL:  0,
		httpClient: &MockHTTPClient{
			StatusCode:   http.StatusOK,
			ResponseBody: data,
		},
		config: &config.Config{CacheDuration: "1h"},
	}

	err := processor.downloadAndProcessData()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(processor.cache) == 0 {
		t.Fatalf("expected cache to be populated")
	}
	if got := processor.cache["US"]; !reflect.DeepEqual(got, []string{"192.168.0.0/24"}) {
		t.Fatalf("US cache = %#v, want %#v", got, []string{"192.168.0.0/24"})
	}
	if got := processor.cache["DE"]; !reflect.DeepEqual(got, []string{"10.0.0.0/16"}) {
		t.Fatalf("DE cache = %#v, want %#v", got, []string{"10.0.0.0/16"})
	}
}

func TestValidateIPCIDR(t *testing.T) {
	testCases := []struct {
		name    string
		cidr    string
		isValid bool
	}{
		{
			name:    "Valid CIDR",
			cidr:    "192.168.1.0/24",
			isValid: true,
		},
		{
			name:    "Valid single IP as CIDR",
			cidr:    "192.168.1.1/32",
			isValid: true,
		},
		{
			name:    "Invalid CIDR - bad mask",
			cidr:    "192.168.1.0/33",
			isValid: false,
		},
		{
			name:    "Invalid CIDR - bad format",
			cidr:    "192.168.1/24",
			isValid: false,
		},
		{
			name:    "Invalid CIDR - no mask",
			cidr:    "192.168.1.0",
			isValid: false,
		},
		{
			name:    "Invalid CIDR - bad IP",
			cidr:    "999.168.1.0/24",
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateIPCIDR(tc.cidr)
			if tc.isValid && err != nil {
				t.Errorf("Expected valid CIDR but got error: %v", err)
			}
			if !tc.isValid && err == nil {
				t.Errorf("Expected error for invalid CIDR but got nil")
			}
		})
	}
}

// TestGetIPListForCountry tests the cache behavior of GetIPListForCountry
func TestGetIPListForCountry(t *testing.T) {
	// Save original args and restore them after test
	origArgs := os.Args
	os.Args = []string{"app"}
	defer func() { os.Args = origArgs }()

	// Create a test processor
	processor := createTestProcessor()
	processor.cacheTTL = 100 * time.Millisecond // Set a shorter TTL for testing

	// Add test data to cache and set cache time to now (valid cache)
	testIPList := []string{"192.168.1.0/24", "10.0.0.0/8"}
	processor.cache["US"] = testIPList
	processor.cacheTime = time.Now() // Set cache time to now so cache is valid

	// Test getting data from cache
	result, err := processor.GetIPListForCountry("US")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, testIPList) {
		t.Errorf("Expected %v, got %v", testIPList, result)
	}

	// Test empty result for non-existent country (with valid cache)
	result, err = processor.GetIPListForCountry("XX")
	if err != nil {
		t.Errorf("Unexpected error for non-existent country: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for non-existent country, got %v", result)
	}

	// Test cache expiration
	// Set cache time to the past so it's expired
	processor.cacheTime = time.Now().Add(-1 * time.Hour)

	// This test is a bit tricky because we'd need to mock the HTTP call
	// For now, we'll just verify that a downloadAndProcessData call is attempted
	// by creating a processor with a non-existent URL to cause a download error
	processor = createTestProcessor()
	processor.cacheTime = time.Now().Add(-1 * time.Hour) // Set an old cache time

	// Test error case when download fails
	// We're expecting an error here since we can't download from a real URL in the test
	// but this at least verifies the download path is attempted
	_, err = processor.GetIPListForCountry("US")
	// We don't check the specific error as it might change based on network conditions
	// Just verify that some error was returned, indicating the download path was attempted
	if err == nil {
		t.Error("Expected error on download attempt, but got nil")
	}
}

func TestParseIPList(t *testing.T) {
	// Sample RIPE data format for testing
	sampleData := `#2.0|ripencc|20220101|123456|+0100
ripencc|US|ipv4|192.168.0.0|256|20220101|allocated
ripencc|DE|ipv4|10.0.0.0|65536|20220101|allocated
# This is a comment line
ripencc|FR|ipv6|2001:db8::|1|20220101|allocated
apnic|CN|ipv4|172.16.0.0|4096|20220101|allocated`

	// Parse the data
	ipDataByCountry, err := parseIPList([]byte(sampleData))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify the results
	if len(ipDataByCountry) != 2 {
		t.Errorf("Expected 2 countries (US, DE), got %d", len(ipDataByCountry))
	}

	// Check the US data
	usData := ipDataByCountry["US"]
	if len(usData) != 1 {
		t.Errorf("Expected 1 IP range for US, got %d", len(usData))
	}
	if usData[0].IPStart != "192.168.0.0" {
		t.Errorf("Expected IP 192.168.0.0, got %s", usData[0].IPStart)
	}
	if usData[0].Count != 256 {
		t.Errorf("Expected count 256, got %d", usData[0].Count)
	}
	if usData[0].CIDRMask != 24 {
		t.Errorf("Expected CIDR mask 24, got %d", usData[0].CIDRMask)
	}

	// Check the DE data
	deData := ipDataByCountry["DE"]
	if len(deData) != 1 {
		t.Errorf("Expected 1 IP range for DE, got %d", len(deData))
	}
	if deData[0].IPStart != "10.0.0.0" {
		t.Errorf("Expected IP 10.0.0.0, got %s", deData[0].IPStart)
	}
	if deData[0].Count != 65536 {
		t.Errorf("Expected count 65536, got %d", deData[0].Count)
	}
	if deData[0].CIDRMask != 16 {
		t.Errorf("Expected CIDR mask 16, got %d", deData[0].CIDRMask)
	}

	// Verify that we didn't parse IPv6 or data from other registries
	if _, ok := ipDataByCountry["FR"]; ok {
		t.Error("Should not have parsed IPv6 data for FR")
	}
	if _, ok := ipDataByCountry["CN"]; ok {
		t.Error("Should not have parsed data from APNIC for CN")
	}
}

// Helper function to parse IP list data - simplified version of the processor's code
func parseIPList(data []byte) (map[string][]IPData, error) {
	ipDataByCountry := make(map[string][]IPData)
	lines := string(data)

	for _, line := range strings.Split(lines, "\n") {
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		// We're only interested in IPv4 records from RIPE NCC
		if parts[0] == "ripencc" && parts[2] == "ipv4" {
			country := strings.ToUpper(parts[1])
			ipStart := parts[3]
			countStr := parts[4]

			count, err := strconv.Atoi(countStr)
			if err != nil {
				continue
			}

			// Calculate CIDR mask from IP count
			mask := 32 - int(math.Log2(float64(count)))

			ipDataByCountry[country] = append(ipDataByCountry[country], IPData{
				Country:  country,
				IPStart:  ipStart,
				Count:    count,
				CIDRMask: mask,
			})
		}
	}

	return ipDataByCountry, nil
}

func TestGetIPListForCountryCaseInsensitive(t *testing.T) {
	// Create a test processor with cached data
	processor := createTestProcessor()
	processor.cacheTTL = 1 * time.Hour

	// Add test data to cache
	testIPList := []string{"192.168.1.0/24", "10.0.0.0/8"}
	processor.cache["US"] = testIPList
	processor.cacheTime = time.Now()

	// Test with lowercase country code
	result, err := processor.GetIPListForCountry("us")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, testIPList) {
		t.Errorf("Expected %v, got %v", testIPList, result)
	}

	// Test with mixed case
	result, err = processor.GetIPListForCountry("Us")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, testIPList) {
		t.Errorf("Expected %v, got %v", testIPList, result)
	}
}

func TestGetIPListForCountryWithMockDownload(t *testing.T) {
	// Sample RIPE data
	mockData := `#2.0|ripencc|20220101|123456|+0100
ripencc|DE|ipv4|192.168.0.0|256|20220101|allocated
ripencc|DE|ipv4|10.0.0.0|65536|20220101|allocated`

	processor := createTestProcessorWithMockData(mockData)
	processor.cacheTime = time.Time{} // Ensure cache is expired

	// Get data for DE
	result, err := processor.GetIPListForCountry("DE")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 CIDRs for DE, got %d", len(result))
	}

	// Verify the CIDRs
	expectedCIDRs := []string{"192.168.0.0/24", "10.0.0.0/16"}
	for _, expected := range expectedCIDRs {
		found := false
		for _, cidr := range result {
			if cidr == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected CIDR %s not found in result %v", expected, result)
		}
	}
}

func TestGetIPListForCountryHTTPError(t *testing.T) {
	processor := createTestProcessor()
	processor.cacheTime = time.Time{} // Ensure cache is expired

	// Try to get data - should fail because of mock error
	_, err := processor.GetIPListForCountry("US")
	if err == nil {
		t.Error("Expected error when HTTP request fails")
	}

	if !strings.Contains(err.Error(), "failed to download") {
		t.Errorf("Error should mention download failure, got: %v", err)
	}
}

func TestGetIPListForCountryNon200Response(t *testing.T) {
	cfg := &config.Config{
		ServerPort:    "8080",
		AuthToken:     "test-token",
		CacheDuration: "1h",
	}

	processor := &Processor{
		cache:     make(map[string][]string),
		cacheTime: time.Time{},
		config:    cfg,
		cacheTTL:  1 * time.Hour,
		httpClient: &MockHTTPClient{
			ShouldError:  false,
			ResponseBody: "Not Found",
			StatusCode:   http.StatusNotFound,
		},
	}

	_, err := processor.GetIPListForCountry("US")
	if err == nil {
		t.Error("Expected error for non-200 response")
	}

	if !strings.Contains(err.Error(), "non-200") {
		t.Errorf("Error should mention non-200 response, got: %v", err)
	}
}

func TestCacheBehavior(t *testing.T) {
	// Create processor with short TTL
	processor := createTestProcessor()
	processor.cacheTTL = 100 * time.Millisecond

	// Add data to cache
	testIPList := []string{"192.168.1.0/24"}
	processor.cache["US"] = testIPList
	processor.cacheTime = time.Now()

	// Should get data from cache
	result, err := processor.GetIPListForCountry("US")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, testIPList) {
		t.Errorf("Expected %v, got %v", testIPList, result)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Now it should try to download (and fail with our mock)
	_, err = processor.GetIPListForCountry("US")
	if err == nil {
		t.Error("Expected error after cache expiration with failing mock")
	}
}

func TestIPDataStruct(t *testing.T) {
	ipData := IPData{
		Country:  "US",
		IPStart:  "192.168.0.0",
		Count:    256,
		CIDRMask: 24,
	}

	if ipData.Country != "US" {
		t.Errorf("Country = %q, want %q", ipData.Country, "US")
	}

	if ipData.IPStart != "192.168.0.0" {
		t.Errorf("IPStart = %q, want %q", ipData.IPStart, "192.168.0.0")
	}

	if ipData.Count != 256 {
		t.Errorf("Count = %d, want %d", ipData.Count, 256)
	}

	if ipData.CIDRMask != 24 {
		t.Errorf("CIDRMask = %d, want %d", ipData.CIDRMask, 24)
	}
}

func TestCIDRMaskCalculation(t *testing.T) {
	testCases := []struct {
		count        int
		expectedMask int
	}{
		{count: 1, expectedMask: 32},
		{count: 2, expectedMask: 31},
		{count: 4, expectedMask: 30},
		{count: 8, expectedMask: 29},
		{count: 16, expectedMask: 28},
		{count: 32, expectedMask: 27},
		{count: 64, expectedMask: 26},
		{count: 128, expectedMask: 25},
		{count: 256, expectedMask: 24},
		{count: 512, expectedMask: 23},
		{count: 1024, expectedMask: 22},
		{count: 65536, expectedMask: 16},
		{count: 16777216, expectedMask: 8},
	}

	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.count), func(t *testing.T) {
			mask := 32 - int(math.Log2(float64(tc.count)))
			if mask != tc.expectedMask {
				t.Errorf("For count %d, expected mask %d, got %d", tc.count, tc.expectedMask, mask)
			}
		})
	}
}

func TestProcessorImplementsInterface(t *testing.T) {
	// This test ensures Processor implements IPProcessor interface
	var _ IPProcessor = (*Processor)(nil)
}
