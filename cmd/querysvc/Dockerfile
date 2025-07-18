# Build
FROM golang:1.24.5-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o querysvc cmd/querysvc/main.go

# Run
FROM alpine:3.17
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/querysvc /usr/local/bin/querysvc
COPY deploy/querysvc/config.yml /etc/querysvc/config.yml:ro

EXPOSE 8082
ENTRYPOINT ["querysvc"]