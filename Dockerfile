# Use scratch for minimal image size - static binary doesn't need OS
FROM scratch

# Copy the pre-built binary from GoReleaser context
COPY antimoji /antimoji

# Set entrypoint
ENTRYPOINT ["/antimoji"]

# Add labels
LABEL org.opencontainers.image.title="Antimoji"
LABEL org.opencontainers.image.description="High-performance emoji detection and removal CLI tool"
LABEL org.opencontainers.image.url="https://github.com/antimoji/antimoji"
LABEL org.opencontainers.image.source="https://github.com/antimoji/antimoji"
LABEL org.opencontainers.image.licenses="MIT"
