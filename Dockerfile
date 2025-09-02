# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/antimoji/antimoji/cmd/antimoji.version=${VERSION:-dev}" \
    -o antimoji ./cmd/antimoji

# Final stage
FROM scratch

# Copy ca-certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/antimoji /antimoji

# Set entrypoint
ENTRYPOINT ["/antimoji"]

# Add labels
LABEL org.opencontainers.image.title="Antimoji"
LABEL org.opencontainers.image.description="High-performance emoji detection and removal CLI tool"
LABEL org.opencontainers.image.url="https://github.com/antimoji/antimoji"
LABEL org.opencontainers.image.source="https://github.com/antimoji/antimoji"
LABEL org.opencontainers.image.licenses="MIT"
