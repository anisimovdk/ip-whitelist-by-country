package config

import (
	"fmt"
	"io"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/anisimovdk/ip-whitelist-by-country/internal/version"
)

var (
	osExit           = os.Exit
	stdOut io.Writer = os.Stdout
)

// Config represents the application configuration
type Config struct {
	ServerPort    string `arg:"--port,env:SERVER_PORT" help:"Port to run the server on"`
	AuthToken     string `arg:"--auth-token,env:AUTH_TOKEN" help:"Authentication token for API requests (leave empty to disable auth)"`
	CacheDuration string `arg:"--cache-duration,env:CACHE_DURATION" help:"Duration to cache IP data (e.g., 24h)"`
	ShowVersion   bool   `arg:"--version,-v" help:"Show version information"`
}

// Version returns the version string for go-arg
func (Config) Version() string {
	return version.GetFullVersion()
}

// Description returns the program description for go-arg
func (Config) Description() string {
	return "IP Whitelist by Country - A service that provides IP network lists filtered by country"
}

// NewConfig parses command-line arguments and returns a Config instance
func NewConfig() *Config {
	cfg := &Config{
		ServerPort:    "8080",
		AuthToken:     "", // Empty by default = no authentication required
		CacheDuration: "1h",
	}

	parser := arg.MustParse(cfg)

	// Handle version flag manually if needed
	if cfg.ShowVersion {
		fmt.Fprintln(stdOut, version.GetFullVersion())
		osExit(0)
	}

	// Add custom help behavior if needed
	_ = parser

	return cfg
}
