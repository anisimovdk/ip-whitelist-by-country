# IP Whitelist by Country

A lightweight Go service that provides IP address ranges (CIDR blocks) for any country based on official RIPE NCC regional internet registry data. Suitable for implementing geo-based access control, firewall rules, or country-specific network policies.

**Key Features:**

- 🌍 IP ranges for all countries from RIPE NCC registry
- ⚡ In-memory caching with configurable TTL
- 🔒 Optional token-based authentication
- 🐳 Pre-built images for amd64, arm64, and armv7

## Table of Contents

- [IP Whitelist by Country](#ip-whitelist-by-country)
  - [Table of Contents](#table-of-contents)
  - [Installation](#installation)
    - [Pre-built Binaries](#pre-built-binaries)
    - [Docker](#docker)
      - [Docker Environment Variables](#docker-environment-variables)
  - [Usage](#usage)
  - [Development](#development)
    - [Prerequisites](#prerequisites)
    - [Running from Source](#running-from-source)
    - [Building from Source](#building-from-source)
    - [Running Tests](#running-tests)
    - [Testing Approach (Design for Testability)](#testing-approach-design-for-testability)
    - [Building Docker Images Locally](#building-docker-images-locally)
      - [Multi-Architecture Builds](#multi-architecture-builds)
  - [CI/CD](#cicd)
    - [Automated Release Process](#automated-release-process)
  - [License](#license)
  - [Disclaimer](#disclaimer)

## Installation

### Pre-built Binaries

Download the latest release from the [GitHub Releases](https://github.com/anisimovdk/ip-whitelist-by-country/releases) page.

Each release includes versioned binaries for Linux platforms:

- `ip-whitelist-linux-amd64-v1.2.3` - Linux x86_64
- `ip-whitelist-linux-arm64-v1.2.3` - Linux ARM64
- `ip-whitelist-linux-armv7-v1.2.3` - Linux ARM v7
- `checksums.txt` - SHA256 checksums for verification

Download and verify:

```bash
# Download binary and checksums
wget https://github.com/anisimovdk/ip-whitelist-by-country/releases/download/v1.2.3/ip-whitelist-linux-amd64-v1.2.3
wget https://github.com/anisimovdk/ip-whitelist-by-country/releases/download/v1.2.3/checksums.txt

# Verify checksum
sha256sum -c --ignore-missing checksums.txt

# Make executable
chmod +x ip-whitelist-linux-amd64-v1.2.3

# Run
./ip-whitelist-linux-amd64-v1.2.3 --version
```

### Docker

Multi-architecture Docker images are automatically published to Docker Hub:

```bash
# Pull the latest release
docker pull anisimovdk/ip-whitelist-by-country:latest

# Pull a specific version
docker pull anisimovdk/ip-whitelist-by-country:v1.2.3

# Run
docker run -p 8080:8080 anisimovdk/ip-whitelist-by-country:v1.2.3
```

Supported architectures:

- `linux/amd64` - x86_64
- `linux/arm64` - ARM 64-bit
- `linux/arm/v7` - ARM 32-bit

Docker will automatically pull the correct architecture for your platform.

#### Docker Environment Variables

Configure the application using environment variables:

- `PORT` - Server port (default: 8080)
- `AUTH_TOKEN` - Authentication token (optional)
- `CACHE_DURATION` - Cache duration (default: 1h)

Example:

```bash
docker run -p 8080:8080 \
  -e PORT=8080 \
  -e AUTH_TOKEN=your-secret-token \
  -e CACHE_DURATION=30m \
  anisimovdk/ip-whitelist-by-country:latest
```

## Usage

The application exposes a REST API:

- `GET /` - Returns a status message
- `GET /get?country=XX&auth=your-token` - Returns a list of IP networks for the specified country code

Example:

```bash
curl "http://localhost:8080/get?country=us&auth=your-token"
```

If authentication is disabled (empty auth token), you can omit the auth parameter:

```bash
curl "http://localhost:8080/get?country=us"
```

## Development

### Prerequisites

- Go 1.26 or later

### Running from Source

```bash
go run cmd/app/main.go
```

Or with command-line arguments:

```bash
go run cmd/app/main.go --port=8080 --auth-token=your-token --cache-duration=1h
```

### Building from Source

```bash
go build -o ip-whitelist cmd/app/main.go
```

Build for all platforms:

```bash
make release-build

# Binaries will be in release/ directory
ls -la release/
```

### Running Tests

To run all tests:

```bash
go test ./...
```

To run tests with coverage:

```bash
go test -cover ./...
```

For a detailed coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

To enforce full coverage locally or in CI:

```bash
make test-cover-100
```

### Testing Approach (Design for Testability)

Some parts of a Go program are traditionally hard to unit test because they have process-wide side effects (e.g., `os.Exit`, `signal.Notify`, binding real network ports, global `http.DefaultServeMux`).

This repository uses a small, idiomatic "test seam" pattern to keep production behavior the same while allowing deterministic unit tests:

- In `internal/config`, the exit and output behaviors are routed through package-level variables (defaulting to `os.Exit` and `os.Stdout`) so tests can safely exercise the `--version` path.
- In `cmd/app`, the main wiring uses package-level function variables (defaulting to the real constructors and stdlib functions) so tests can stub server startup and signal handling without touching real ports or OS signals.
- In `internal/handler`, routes can be registered on a provided `http.ServeMux` to avoid cross-test conflicts on the global mux.

### Building Docker Images Locally

Build the Docker image:

```bash
make docker-build
# or manually:
docker build -t ip-whitelist:latest .
```

Run the container:

```bash
make docker-run
# or manually:
docker run -p 8080:8080 --name ip-whitelist ip-whitelist:latest
```

Stop the container:

```bash
make docker-stop
```

Clean up Docker resources:

```bash
make docker-clean
```

#### Multi-Architecture Builds

The Dockerfile supports building for multiple architectures using Docker buildx:

Setup buildx (one-time setup):

```bash
make docker-buildx-setup
```

Build for multiple architectures:

```bash
make docker-build-multiarch
```

Build for a specific architecture:

```bash
make docker-build-amd64
make docker-build-arm64
make docker-build-armv7
```

## CI/CD

This project uses [Argo Workflows](https://argoproj.github.io/workflows/) and [Argo Events](https://argoproj.github.io/events/) for continuous integration and deployment, configured via the [argo-ci-charts](https://github.com/anisimovdk/argo-ci-charts) Helm charts.

### Automated Release Process

Releases are triggered automatically when you push a version tag:

```bash
# Create and push a version tag
git tag v1.2.3
git push origin v1.2.3
```

The CI pipeline will:

1. **Clone** the repository at the tagged commit
2. **Test** - run `make ci` (mod-tidy, fmt, vet, test-verbose, test-race, test-cover)
3. **Build** - compile binaries for all platforms:
   - `ip-whitelist-linux-amd64-v1.2.3`
   - `ip-whitelist-linux-arm64-v1.2.3`
   - `ip-whitelist-linux-armv7-v1.2.3`
4. **GitHub Release** - create a release with binaries and checksums
5. **Docker Build & Push** - build and push multi-arch images to Docker Hub

All steps run in parallel after tests pass. The Docker image contains the **exact same binaries** as the GitHub release (no recompilation).

## License

Apache 2.0

## Disclaimer

THIS SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

The IP address data is sourced from the RIPE NCC registry. While we strive to provide accurate and up-to-date information, we make no guarantees about the completeness, accuracy, or reliability of this data. Use at your own risk.
