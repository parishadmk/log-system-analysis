# Log-System Analysis

A fully scalable log analysis system using **Kafka**, **Cassandra**, **ClickHouse**, and **CockroachDB**, with a Go backend and a React+Vite frontend.

---

## Architecture Overview

* **AuthSvc** (`cmd/authsvc`): gRPC + HTTP façade for user login and API-key validation (CockroachDB).
* **Gateway** (`cmd/gateway`): HTTP ingestion endpoint (`/v1/logs`), validates API-key via AuthSvc, publishes raw logs to Kafka.
* **Processor** (`cmd/processor`): Kafka consumer, dual-writes to Cassandra (raw events, TTL) and ClickHouse (analytics).
* **QuerySvc** (`cmd/querysvc`): HTTP endpoints (`/v1/search`, `/v1/detail`), JWT-protected, queries ClickHouse and Cassandra.
* **Frontend** (`frontend/`): React+Vite UI for login, project list, event search, and detail view.
* **Infra** (via Docker Compose): ZooKeeper, Kafka, Cassandra, ClickHouse, CockroachDB, Prometheus, Grafana.

---

## Prerequisites

* **Docker Desktop** (to run containers)
* **docker-compose** (v2+)
* **Go** (1.24+; for codegen)
* **protoc** + **protoc-gen-go**, **protoc-gen-go-grpc** (for `.proto` stubs)
* **Node.js** & **npm** (for frontend)
* **Homebrew** (macOS) to install missing CLIs:

  ```bash
  brew install go node protoc grpcurl
  ```

---

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/parishadmk/log-system-analysis.git
cd log-system-analysis
```

### 2. Generate gRPC stubs & Go modules

```bash
go mod init github.com/parishadmk/log-system-analysis
go mod tidy
make proto   # builds .pb.go files into internal/api/{auth,ingest,query}
```

### 3. Bring up infrastructure + services

```bash
# Starts all infra containers and services: AuthSvc, Gateway, Processor, QuerySvc
./scripts/local-up.sh
```

Verify with:

```bash
docker-compose ps
```

You should see services **Up** on ports:

* **8080**: AuthSvc (gRPC + HTTP `/v1/auth/login`)
* **8081**: Gateway
* **9100**: Processor metrics
* **8082**: QuerySvc
* **3000**: Frontend

---

### 4. Apply database migrations

#### CockroachDB (users & projects)

```bash
docker exec -i log-system-analysis-cockroach-1 \
  cockroach sql --insecure --host=localhost:26257 \
  < deploy/migrations/001_create_users_projects.sql
```

#### Cassandra (raw events)

```bash
docker exec -i log-system-analysis-cassandra-1 \
  cqlsh cassandra 9042 < deploy/migrations/002_create_cassandra_logs.cql
```

#### ClickHouse (analytics table)

```bash
docker exec -i log-system-analysis-clickhouse-1 \
  clickhouse-client --multiquery < deploy/migrations/003_create_clickhouse_logs.sql
```

---

### 5. Create test data

#### Create a demo user

Generate a bcrypt hash for password:

```bash
go run scripts/hash_password.go  # prints $2a$...HASH
```

Insert into CockroachDB:

```bash
docker exec -i log-system-analysis-cockroach-1 \
  cockroach sql --insecure --host=localhost:26257 <<'EOF'
INSERT INTO users (username, hashed_password)
VALUES ('alice', '$2a$...HASH');
EOF
```

#### Create a demo project

```bash
docker exec -i log-system-analysis-cockroach-1 \
  cockroach sql --insecure --host=localhost:26257 <<'EOF'
INSERT INTO projects (name, searchable_keys, api_key, ttl_days)
VALUES ('demo_project', ARRAY['foo','bar'], 'my-demo-api-key-123', 30);
EOF
```

Retrieve the `id`:

```bash
docker exec -i log-system-analysis-cockroach-1 \
  cockroach sql --insecure --host=localhost:26257 -e \
"SELECT id, api_key FROM projects WHERE name='demo_project';"
```

---

### 6. Test ingestion end-to-end

```bash
# Prepare JSON payload (in host shell)
PROJECT_ID=<paste-uuid>
API_KEY="my-demo-api-key-123"
TS=$(( $(date +%s) * 1000000000 ))
cat > payload.json <<EOF
{
  "project_id":"$PROJECT_ID",
  "api_key":"$API_KEY",
  "payload":{ "name":"test_event","timestamp":$TS,"data":{"foo":"bar"} }
}
EOF

# Send to Gateway
curl -v -H "Content-Type: application/json" --data-binary @payload.json \
  http://localhost:8081/v1/logs
# Expect HTTP/1.1 202 Accepted

# Consume from Kafka
docker exec -i log-system-analysis-kafka-1 bash -c "\
  kafka-console-consumer --bootstrap-server localhost:9092 --topic logs_raw \
    --from-beginning --max-messages 1 --property print.key=true\"
```

---

### 7. Test Processor writes

```bash
# Cassandra
docker exec -i log-system-analysis-cassandra-1 \
  cqlsh cassandra 9042 -e "SELECT project_id, event_name, data FROM logs.events LIMIT 1;"
# ClickHouse
docker exec -i log-system-analysis-clickhouse-1 \
  clickhouse-client --query="SELECT project_id, event_name, data FROM logs LIMIT 1;"
```

---

### 8. Test QuerySvc endpoints

#### Obtain JWT via HTTP

```bash
curl -v -X POST http://localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"secret"}'
# => { "token": "<JWT>" }
export TOKEN="<JWT>"
```

#### Search events

```bash
curl -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"'$PROJECT_ID'","filters":{}}' \
  http://localhost:8082/v1/search
```

#### Event detail

```bash
curl -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"'$PROJECT_ID'","event_name":"test_event"}' \
  http://localhost:8082/v1/detail
```

---

## Frontend (React + Vite)

1. **Install dependencies**

   ```bash
   cd frontend
   npm install
   ```

2. **Run dev server**

   ```bash
   npm run dev
   ```

   * Open [http://localhost:3000](http://localhost:3000) in your browser.

3. **Login**

   * **Username:** `alice`
   * **Password:** `secret`

4. **Navigate**

   * **Dashboard**: select **demo\_project**.
   * **Search**: view table of event names, counts, timestamps.
   * **Detail**: click an event to view its payload and navigate next/prev.

5. **Build for production**

   ```bash
   npm run build
   ```

---

## Dashboards & Monitoring

* **Prometheus**: [http://localhost:9090](http://localhost:9090)
* **Grafana**: [http://localhost:3000](http://localhost:3000) (default admin\:admin)
* **Processor metrics**: [http://localhost:9100/metrics](http://localhost:9100/metrics)

---

## Cleanup

```bash
docker-compose down -v
```

---

## Notes

* Update `deploy/*/config.yml` with real secrets in production.
* Use TLS and proper auth for all services when deploying.
* Extend QuerySvc with pagination cursors and project-level authorization as needed.
