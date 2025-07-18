package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go"
	"github.com/gocql/gocql"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/parishadmk/log-system-analysis/internal/lib"
)

type config struct {
	Cockroach  struct{ Dsn string }
	ClickHouse struct{ Dsn string }
	Server     struct {
		Port   string
		JwtKey string
	}
}

var zapLog *zap.Logger

// JWT claims
type claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// Search request/response
type SearchRequest struct {
	ProjectID string            `json:"project_id"`
	Filters   map[string]string `json:"filters"`
}
type EventSummary struct {
	Name     string `json:"name"`
	LastSeen int64  `json:"last_seen"`
	Count    int64  `json:"count"`
}
type SearchResponse struct {
	Events []EventSummary `json:"events"`
}

// Detail request/response
type DetailRequest struct {
	ProjectID string `json:"project_id"`
	EventName string `json:"event_name"`
	Cursor    string `json:"cursor,omitempty"`
}
type DetailResponse struct {
	Cursor string                 `json:"cursor"`
	Entry  map[string]interface{} `json:"entry"`
}

func main() {
	// 1) Logger & config
	if err := lib.InitLogger(); err != nil {
		panic(err)
	}
	zapLog = lib.Log
	viper.SetConfigFile("/etc/querysvc/config.yml")
	if err := lib.LoadConfig(viper.ConfigFileUsed()); err != nil {
		zapLog.Fatal("config load", zap.Error(err))
	}
	var cfg config
	if err := viper.Unmarshal(&cfg); err != nil {
		zapLog.Fatal("config unmarshal", zap.Error(err))
	}

	// 2) DB connections
	// Cockroach (for JWT project membership, if implemented later)
	crdb, err := lib.NewCockroachPool(cfg.Cockroach.Dsn)
	if err != nil {
		zapLog.Fatal("cockroach connect", zap.Error(err))
	}
	defer crdb.Close()

	// ClickHouse (search)
	chDB, err := sql.Open("clickhouse", cfg.ClickHouse.Dsn)
	if err != nil {
		zapLog.Fatal("clickhouse connect", zap.Error(err))
	}
	defer chDB.Close()

	// Cassandra (detail)
	cassSess, err := lib.NewCassandraSession([]string{"cassandra:9042"})
	if err != nil {
		zapLog.Fatal("cassandra connect", zap.Error(err))
	}
	defer cassSess.Close()

	// 3) HTTP handlers + middleware
	mux := http.NewServeMux()
	mux.Handle("/v1/search", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searchHandler(w, r, chDB)
	})))
	mux.Handle("/v1/detail", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detailHandler(w, r, cassSess)
	})))

	// 4) Start HTTP server
	zapLog.Info("QuerySvc listening", zap.String("port", cfg.Server.Port))
	if err := http.ListenAndServe(":"+cfg.Server.Port, mux); err != nil {
		zapLog.Fatal("HTTP serve failed", zap.Error(err))
	}
}

// authMiddleware verifies the JWT in Authorization: Bearer <token>
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(hdr, "Bearer ")
		jwtKey := []byte(viper.GetString("server.jwt_key"))
		token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(t *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		// could extract user_id from claims here if needed
		next.ServeHTTP(w, r)
	})
}

// searchHandler queries ClickHouse for event summaries
func searchHandler(w http.ResponseWriter, r *http.Request, chDB *sql.DB) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		zapLog.Warn("search decode", zap.Error(err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// build SQL
	sqlStr := `SELECT event_name, max(timestamp) AS last_seen, count() AS cnt
               FROM logs
               WHERE project_id = ?`
	args := []interface{}{req.ProjectID}
	for k, v := range req.Filters {
		sqlStr += fmt.Sprintf(" AND data['%s']=?", k)
		args = append(args, v)
	}
	sqlStr += " GROUP BY event_name"

	rows, err := chDB.Query(sqlStr, args...)
	if err != nil {
		zapLog.Error("clickhouse search", zap.Error(err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var resp SearchResponse
	for rows.Next() {
		var ev EventSummary
		if err := rows.Scan(&ev.Name, &ev.LastSeen, &ev.Count); err != nil {
			zapLog.Error("row scan", zap.Error(err))
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		resp.Events = append(resp.Events, ev)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// detailHandler retrieves one event from Cassandra
func detailHandler(w http.ResponseWriter, r *http.Request, cassSess *gocql.Session) {
	var req DetailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		zapLog.Warn("detail decode", zap.Error(err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// fetch the latest event_name instance
	// NOTE: using ALLOW FILTERING for simplicity; in prod you'd maintain a secondary index
	cql := `
      SELECT event_time, event_name, data 
        FROM logs.events 
       WHERE project_id=? AND event_name=? 
       ORDER BY kafka_partition DESC, kafka_offset DESC
       LIMIT 1 ALLOW FILTERING`
	iter := cassSess.Query(cql, req.ProjectID, req.EventName).Iter()
	var evTime time.Time
	var name string
	var data map[string]string
	if !iter.Scan(&evTime, &name, &data) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := iter.Close(); err != nil {
		zapLog.Error("cassandra iter", zap.Error(err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// build the JSON entry
	entry := map[string]interface{}{
		"event_time": evTime.UnixNano(),
		"event_name": name,
		"data":       data,
	}
	// use timestamp nanosecond as cursor
	cursor := fmt.Sprintf("%d", evTime.UnixNano())

	resp := DetailResponse{Cursor: cursor, Entry: entry}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
