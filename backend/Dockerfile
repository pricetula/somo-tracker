FROM golang:1.26-alpine AS builder

RUN go install github.com/air-verse/air@latest

FROM golang:1.26-alpine

RUN apk add --no-cache git ca-certificates tzdata

COPY --from=builder /go/bin/air /usr/local/bin/air

WORKDIR /app

CMD ["air", "-c", ".air.toml"]
