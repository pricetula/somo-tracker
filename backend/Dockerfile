# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache --update git ca-certificates tzdata

WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build a statically linked binary
# - ldflags: strip debug info and disable DWARF for smallest size
# - CGO_ENABLED=0: fully static binary (no libc dependency)
# - GOOS=linux GOARCH=amd64: explicit for Render's amd64 instances
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /build/api \
    ./cmd/api

# ── Stage 2: Runtime ────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache --update ca-certificates tzdata wget

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binary
COPY --from=builder /build/api .

# Copy migrations (golang-migrate loads them via file:// path relative to CWD)
COPY --from=builder /src/internal/database/migrations ./internal/database/migrations

# Use the non-root user
USER appuser

# HEALTHCHECK — probes the HTTP health endpoint.
# Render uses this for zero-downtime deploys and early restarts.
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:${PORT:-3030}/health || exit 1

# The app reads PORT from env; Render sets it automatically.
# Default fallback in config.go is 3030.
EXPOSE 3030

ENTRYPOINT ["./api"]
