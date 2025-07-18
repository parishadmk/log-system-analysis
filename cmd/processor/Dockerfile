# Build stage
FROM golang:1.24.5-alpine AS builder
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source & build
COPY . .
RUN go build -o processor cmd/processor/main.go

# Final image
FROM alpine:3.17
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/processor /usr/local/bin/processor
COPY deploy/processor/config.yml /etc/processor/config.yml:ro

EXPOSE 9100
ENTRYPOINT ["processor"]