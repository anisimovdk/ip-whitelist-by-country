# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build arguments for cross-compilation
ARG TARGETOS
ARG TARGETARCH

# Version information build arguments
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG GO_VERSION=unknown

# Build the binary with optimizations and version information for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -extldflags \"-static\" \
    -X 'github.com/anisimovdk/ip-whitelist-by-country/internal/version.Version=${VERSION}' \
    -X 'github.com/anisimovdk/ip-whitelist-by-country/internal/version.GitCommit=${GIT_COMMIT}' \
    -X 'github.com/anisimovdk/ip-whitelist-by-country/internal/version.BuildDate=${BUILD_DATE}' \
    -X 'github.com/anisimovdk/ip-whitelist-by-country/internal/version.GoVersion=${GO_VERSION}'" \
    -a -installsuffix cgo \
    -o ip-whitelist \
    ./cmd/app

# Default (source-build) final stage — used for local docker build
FROM alpine

RUN apk add --no-cache ca-certificates

# Create non-root user with GID 0 (root group) for OpenShift/Kubernetes compatibility
RUN adduser -u 10001 -G root -H -D appuser

# Copy version information build arguments to final stage
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG GO_VERSION=unknown

# Add metadata labels
LABEL org.opencontainers.image.title="IP Whitelist by Country"
LABEL org.opencontainers.image.description="A service that provides IP network lists filtered by country"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.source="https://github.com/anisimovdk/ip-whitelist-by-country"
LABEL org.opencontainers.image.url="https://github.com/anisimovdk/ip-whitelist-by-country"
LABEL org.opencontainers.image.documentation="https://github.com/anisimovdk/ip-whitelist-by-country/blob/main/README.md"
LABEL org.opencontainers.image.vendor="anisimovdk"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.ref.name="${VERSION}"
LABEL go.version="${GO_VERSION}"

# Copy the binary
COPY --from=builder /build/ip-whitelist /app/ip-whitelist

# Set ownership and permissions for GID-0 arbitrary-UID pattern
RUN chown -R 10001:0 /app && chmod -R g=u /app

# Expose port
EXPOSE 8080

# Run as non-root user
USER 10001:0

# Set the binary as entrypoint
ENTRYPOINT ["/app/ip-whitelist"]

# Default command arguments
CMD ["--port=8080"]
