# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build frontend
WORKDIR /build/frontend
RUN apk add --no-cache nodejs npm
RUN npm ci && npm run build

# Build backend with embedded frontend
WORKDIR /build
RUN go build \
    -ldflags "-X 'main.buildVersion=$(git describe --tags --always --dirty 2>/dev/null || echo "docker")' -X 'main.buildTime=$(date -u +"%Y-%m-%dT%H:%M:%SZ")'" \
    -o allstar-nexus \
    .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget

# Create app user (UID 1000)
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Create data directory
RUN mkdir -p /app/data && \
    chown -R appuser:appuser /app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/allstar-nexus .
COPY --from=builder /build/config.yaml.example .

# Switch to non-root user
USER 1000

# Expose port
EXPOSE 8080

# Volume for persistent data
# VOLUME ["/app/data"]

# Run the application
CMD ["./allstar-nexus", "--config", "/app/data/config.yaml"]
