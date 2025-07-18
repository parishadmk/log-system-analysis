package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/spf13/viper"
    "go.uber.org/zap"
    "github.com/IBM/sarama"
    "google.golang.org/grpc"
    "google.golang.org/protobuf/proto"

    authpb "github.com/parishadmk/log-system-analysis/internal/api/auth"
    ingestpb "github.com/parishadmk/log-system-analysis/internal/api/ingest"
    "github.com/parishadmk/log-system-analysis/internal/lib"
)

type gatewayServer struct {
    logger     *zap.Logger
    kafkaProd  sarama.SyncProducer
    kafkaTopic string
    authClient authpb.AuthServiceClient
}

// request shape matches ingestpb.LogRequest
type httpLogRequest struct {
    ProjectID string            `json:"project_id"`
    ApiKey    string            `json:"api_key"`
    Payload   ingestpb.LogPayload `json:"payload"`
}

func main() {
    // Init logger
    if err := lib.InitLogger(); err != nil {
        panic(err)
    }
    logger := lib.Log

    // Load config
    viper.SetConfigFile("/etc/gateway/config.yml")
    if err := lib.LoadConfig(viper.ConfigFileUsed()); err != nil {
        logger.Fatal("failed to load config", zap.Error(err))
    }
    brokers := viper.GetStringSlice("kafka.brokers")
    topic := viper.GetString("kafka.topic")
    authAddr := viper.GetString("authsvc.address")
    port := viper.GetString("server.port")

    // Kafka producer
    prod, err := lib.NewKafkaProducer(brokers)
    if err != nil {
        logger.Fatal("kafka producer init failed", zap.Error(err))
    }

    // gRPC auth client
    conn, err := grpc.Dial(authAddr, grpc.WithInsecure())
    if err != nil {
        logger.Fatal("failed to dial authsvc", zap.Error(err))
    }
    authClient := authpb.NewAuthServiceClient(conn)

    srv := &gatewayServer{
        logger:     logger,
        kafkaProd:  prod,
        kafkaTopic: topic,
        authClient: authClient,
    }

    http.HandleFunc("/v1/logs", srv.handleSendLog)
    logger.Info("Gateway listening", zap.String("port", port))
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        logger.Fatal("HTTP server failed", zap.Error(err))
    }
}

func (g *gatewayServer) handleSendLog(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
        return
    }

    var req httpLogRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        g.logger.Warn("invalid JSON", zap.Error(err))
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    // 1) Validate API key
    ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
    defer cancel()
    authResp, err := g.authClient.ValidateApiKey(ctx, &authpb.ApiKeyRequest{
        ProjectId: req.ProjectID,
        ApiKey:    req.ApiKey,
    })
    if err != nil || !authResp.Valid {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // 2) Build protobuf message
    pb := &ingestpb.LogRequest{
        ProjectId: req.ProjectID,
        ApiKey:    req.ApiKey,
        Payload:   &req.Payload,
    }
    data, err := proto.Marshal(pb)
    if err != nil {
        g.logger.Error("proto marshal failed", zap.Error(err))
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }

    // 3) Produce to Kafka (keyed by project for ordering)
    msg := &sarama.ProducerMessage{
        Topic: g.kafkaTopic,
        Key:   sarama.StringEncoder(req.ProjectID),
        Value: sarama.ByteEncoder(data),
    }
    if _, _, err := g.kafkaProd.SendMessage(msg); err != nil {
        g.logger.Error("kafka send failed", zap.Error(err))
        http.Error(w, "server error", http.StatusInternalServerError)
        return
    }

    // 4) Ack
    w.WriteHeader(http.StatusAccepted)
    fmt.Fprint(w, `{"accepted":true}`)
}