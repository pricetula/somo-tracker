# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache --update git ca-certificates tzdata

WORKDIR /app

# Cache module downloads (separate layer so go.mod/go.sum changes don't
# invalidate the full source copy)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a fully static binary — zero libc dependencies
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /build/api \
    ./cmd/api

# ── Stage 2: Runtime ────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache --update ca-certificates tzdata wget

# Create a non-root user for security best practices
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binary
COPY --from=builder /build/api .

# Copy migrations (golang-migrate resolves them via file:// relative to CWD)
COPY --from=builder /app/internal/database/migrations ./internal/database/migrations

# Drop privileges
USER appuser

# HEALTHCHECK — Render uses this for zero-downtime deploys and early restarts
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:${PORT:-3030}/health || exit 1

# Render injects the PORT env var; default fallback in config.go is 3030
EXPOSE 3030

ENTRYPOINT ["./api"]
