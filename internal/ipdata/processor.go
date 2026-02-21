package ipdata

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anisimovdk/ip-whitelist-by-country/internal/config"
)

const (
	downloadTimeout = 60 * time.Second
)

var ripeURL = "https://ftp.ripe.net/ripe/stats/delegated-ripencc-extended-latest"

// HTTPClient interface for making HTTP requests (allows mocking)
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// IPData represents a single IP allocation record
type IPData struct {
	Country  string
	IPStart  string
	Count    int
	CIDRMask int
}

// Processor handles IP data processing
type Processor struct {
	cache      map[string][]string // country code -> list of CIDR blocks
	cacheTime  time.Time
	config     *config.Config
	cacheTTL   time.Duration
	mutex      sync.RWMutex
	httpClient HTTPClient
}

// NewProcessor creates a new processor
func NewProcessor() *Processor {
	cfg := config.NewConfig()
	cacheDuration, err := time.ParseDuration(cfg.CacheDuration)
	if err != nil {
		cacheDuration = 1 * time.Hour // Default to 1 hour if parsing fails
	}

	return &Processor{
		cache:      make(map[string][]string),
		cacheTime:  time.Time{},
		config:     cfg,
		cacheTTL:   cacheDuration,
		httpClient: http.DefaultClient,
	}
}

// NewProcessorWithClient creates a new processor with a custom HTTP client (useful for testing)
func NewProcessorWithClient(httpClient HTTPClient) *Processor {
	cfg := config.NewConfig()
	cacheDuration, err := time.ParseDuration(cfg.CacheDuration)
	if err != nil {
		cacheDuration = 1 * time.Hour // Default to 1 hour if parsing fails
	}

	return &Processor{
		cache:      make(map[string][]string),
		cacheTime:  time.Time{},
		config:     cfg,
		cacheTTL:   cacheDuration,
		httpClient: httpClient,
	}
}

// GetIPListForCountry returns a list of IP CIDR blocks for a country
func (p *Processor) GetIPListForCountry(countryCode string) ([]string, error) {
	countryCode = strings.ToUpper(countryCode)

	// Check cache first
	p.mutex.RLock()
	if time.Since(p.cacheTime) < p.cacheTTL {
		if ipList, ok := p.cache[countryCode]; ok {
			p.mutex.RUnlock()
			return ipList, nil
		}
	}
	p.mutex.RUnlock()

	// Need to download and process data
	err := p.downloadAndProcessData()
	if err != nil {
		return nil, fmt.Errorf("failed to download and process data: %w", err)
	}

	// Check cache again
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if ipList, ok := p.cache[countryCode]; ok {
		return ipList, nil
	}

	return []string{}, nil // Return empty list if country not found
}

// downloadAndProcessData downloads and processes the RIPE data
func (p *Processor) downloadAndProcessData() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check cache again after obtaining write lock
	if time.Since(p.cacheTime) < p.cacheTTL {
		return nil
	}

	log.Println("Downloading IP data from RIPE NCC...")

	// Create context with timeout for the HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ripeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Perform the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	// Process the data
	ipDataByCountry := make(map[string][]IPData)
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		// We're only interested in IPv4 records
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

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Convert to CIDR notation and update cache
	newCache := make(map[string][]string)
	for country, ipDataList := range ipDataByCountry {
		cidrList := make([]string, 0, len(ipDataList))
		for _, ipData := range ipDataList {
			cidr := fmt.Sprintf("%s/%d", ipData.IPStart, ipData.CIDRMask)
			cidrList = append(cidrList, cidr)
		}
		newCache[country] = cidrList
	}

	// Update cache
	p.cache = newCache
	p.cacheTime = time.Now()

	log.Printf("IP data processed. Found data for %d countries\n", len(p.cache))
	return nil
}

// ValidateIPCIDR ensures the IP/CIDR is valid
func ValidateIPCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return errors.New("invalid CIDR notation")
	}
	return nil
}
