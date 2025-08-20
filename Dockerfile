# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build info
ARG VERSION=unknown
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME} -w -s" \
    -a -installsuffix cgo \
    -o pipegen .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 pipegen && \
    adduser -u 1000 -G pipegen -s /bin/sh -D pipegen

# Set working directory
WORKDIR /home/pipegen

# Copy binary from builder stage
COPY --from=builder /app/pipegen /usr/local/bin/pipegen

# Copy web assets if they exist
COPY --from=builder /app/web ./web

# Create directories for data
RUN mkdir -p data logs && \
    chown -R pipegen:pipegen /home/pipegen

# Switch to non-root user
USER pipegen

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pipegen version || exit 1

# Expose common ports (dashboard, kafka, schema registry)
EXPOSE 8080 9092 8081

# Default command
CMD ["pipegen", "help"]

# Labels
LABEL maintainer="PipeGen Team"
LABEL version="${VERSION}"
LABEL description="A powerful CLI tool for creating and managing streaming data pipelines"
