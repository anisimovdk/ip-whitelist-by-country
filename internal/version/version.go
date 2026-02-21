package version

// Version information that can be overridden at build time
var (
	// Version is the semantic version of the application
	Version = "dev"

	// GitCommit is the git commit hash (set by build process)
	GitCommit = "unknown"

	// BuildDate is when the binary was built (set by build process)
	BuildDate = "unknown"

	// GoVersion is the Go version used to build the binary
	GoVersion = "unknown"
)

// GetVersion returns a formatted version string
func GetVersion() string {
	if Version == "dev" {
		return "dev (development build)"
	}
	return Version
}

// GetFullVersion returns a detailed version string with all build information
func GetFullVersion() string {
	return "Version: " + Version + "\n" +
		"Git Commit: " + GitCommit + "\n" +
		"Build Date: " + BuildDate + "\n" +
		"Go Version: " + GoVersion
}
