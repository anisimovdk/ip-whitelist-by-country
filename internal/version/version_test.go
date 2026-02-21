package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "development version",
			version:  "dev",
			expected: "dev (development build)",
		},
		{
			name:     "release version",
			version:  "1.0.0",
			expected: "1.0.0",
		},
		{
			name:     "pre-release version",
			version:  "1.0.0-beta",
			expected: "1.0.0-beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value and restore after test
			origVersion := Version
			defer func() { Version = origVersion }()

			Version = tt.version
			result := GetVersion()

			if result != tt.expected {
				t.Errorf("GetVersion() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetFullVersion(t *testing.T) {
	// Save original values and restore after test
	origVersion := Version
	origGitCommit := GitCommit
	origBuildDate := BuildDate
	origGoVersion := GoVersion
	defer func() {
		Version = origVersion
		GitCommit = origGitCommit
		BuildDate = origBuildDate
		GoVersion = origGoVersion
	}()

	// Set test values
	Version = "1.2.3"
	GitCommit = "abc123"
	BuildDate = "2025-12-23"
	GoVersion = "go1.26"

	result := GetFullVersion()

	// Check all expected components are present
	expectedComponents := []string{
		"Version: 1.2.3",
		"Git Commit: abc123",
		"Build Date: 2025-12-23",
		"Go Version: go1.26",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(result, component) {
			t.Errorf("GetFullVersion() missing %q, got: %s", component, result)
		}
	}
}

func TestGetFullVersionDefault(t *testing.T) {
	// Test with default values (don't modify the variables)
	result := GetFullVersion()

	// Should contain all field labels
	if !strings.Contains(result, "Version:") {
		t.Error("GetFullVersion() missing 'Version:' label")
	}
	if !strings.Contains(result, "Git Commit:") {
		t.Error("GetFullVersion() missing 'Git Commit:' label")
	}
	if !strings.Contains(result, "Build Date:") {
		t.Error("GetFullVersion() missing 'Build Date:' label")
	}
	if !strings.Contains(result, "Go Version:") {
		t.Error("GetFullVersion() missing 'Go Version:' label")
	}
}

func TestVersionVariablesDefaults(t *testing.T) {
	// These tests verify the default values are set correctly
	// Note: These might fail if run after other tests that modify the values
	// In a real scenario, you'd want to run these in isolation or save/restore values

	if Version == "" {
		t.Error("Version should not be empty")
	}

	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}

	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}

	if GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
}
