# Stage 1: Builder
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api/main.go

# Stage 2: Runner
FROM alpine:latest

# Install ca-certificates for HTTPS/secure connections
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy certificates and the compiled binary from the builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/api /bin/api

EXPOSE 3030

ENTRYPOINT ["/bin/api"]
