FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/api ./cmd/api

FROM alpine:latest
RUN apk add --no-cache wget
COPY --from=build /bin/api /bin/api
COPY --from=build /app/migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
