#  Build stage
FROM golang:1.24.5-alpine AS builder
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN go build -o gateway cmd/gateway/main.go

#  Final image
FROM alpine:3.17
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/gateway /usr/local/bin/gateway
COPY deploy/gateway/config.yml /etc/gateway/config.yml:ro

EXPOSE 8081
ENTRYPOINT ["gateway"]