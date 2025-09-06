# Multi-stage build for proper binary compilation
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o antimoji ./cmd/antimoji

# Final stage - minimal runtime image
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/antimoji /antimoji

# Set entrypoint
ENTRYPOINT ["/antimoji"]

# Add labels
LABEL org.opencontainers.image.title="Antimoji"
LABEL org.opencontainers.image.description="High-performance emoji detection and removal CLI tool"
LABEL org.opencontainers.image.url="https://github.com/jamesainslie/antimoji"
LABEL org.opencontainers.image.source="https://github.com/jamesainslie/antimoji"
LABEL org.opencontainers.image.licenses="MIT"
