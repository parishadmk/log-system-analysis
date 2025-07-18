# === Configuration ===
COMPOSE = docker-compose
PROTOC = protoc
GO = go

.PHONY: up down build proto fmt

up:
	@$(COMPOSE) up -d

down:
	@$(COMPOSE) down

build:
	@$(GO) build ./...

proto:
	@echo "🛠️  Building proto stubs..."
	@# ensure output dirs exist
	@mkdir -p internal/api/auth internal/api/ingest internal/api/query

	@# auth.proto → internal/api/auth
	@protoc -I proto \
	  --go_out=paths=source_relative:internal/api/auth \
	  --go-grpc_out=paths=source_relative:internal/api/auth \
	  proto/auth.proto

	@# log_ingest.proto → internal/api/ingest
	@protoc -I proto \
	  --go_out=paths=source_relative:internal/api/ingest \
	  --go-grpc_out=paths=source_relative:internal/api/ingest \
	  proto/log_ingest.proto

	@# query.proto → internal/api/query
	@protoc -I proto \
	  --go_out=paths=source_relative:internal/api/query \
	  --go-grpc_out=paths=source_relative:internal/api/query \
	  proto/query.proto

fmt:
	@$(GO) fmt ./...
