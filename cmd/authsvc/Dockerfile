# Build stage
FROM golang:1.24.5-alpine AS builder
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy code & generated protos
COPY . .

# Build
RUN go build -o authsvc cmd/authsvc/main.go

# Final image
FROM alpine:3.17
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/authsvc /usr/local/bin/authsvc

# Copy config
COPY deploy/authsvc/config.yml /etc/authsvc/config.yml

EXPOSE 8080
ENTRYPOINT ["authsvc"]