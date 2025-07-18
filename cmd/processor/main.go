package main

import (
    "context"
    "database/sql"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/IBM/sarama"
    "github.com/gocql/gocql"
    "github.com/parishadmk/log-system-analysis/internal/lib"
    ingestpb "github.com/parishadmk/log-system-analysis/internal/api/ingest"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/spf13/viper"
    "go.uber.org/zap"
    "google.golang.org/protobuf/proto"
)

type processorConfig struct {
    Kafka struct {
        Brokers []string
        Topic   string
        Group   string
    }
    Cassandra struct {
        Hosts []string
    }
    ClickHouse struct {
        Dsn string
    }
    Processor struct {
        TtlSeconds int
    }
    Metrics struct {
        Port string
    }
}

var (
    processedCounter = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "processor_messages_processed_total",
        Help: "Total number of messages successfully processed",
    })
    errorCounter = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "processor_messages_failed_total",
        Help: "Total number of messages that failed processing",
    })
    writeLatencyHist = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "processor_write_latency_seconds",
        Help: "Latency (s) for dual-write to Cassandra + ClickHouse",
    })
)

func initMetrics() {
    prometheus.MustRegister(processedCounter, errorCounter, writeLatencyHist)
    http.Handle("/metrics", promhttp.Handler())
}

// handler for the Sarama consumer group
type consumerGroupHandler struct {
    logger   *zap.Logger
    cassSess *gocql.Session
    chDB     *sql.DB
    ttl      int
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        start := time.Now()
        if err := h.processMessage(msg); err != nil {
            h.logger.Error("processing failed", zap.Error(err))
            errorCounter.Inc()
            // still mark so we don’t block on a poison message
            sess.MarkMessage(msg, "")
            continue
        }
        writeLatencyHist.Observe(time.Since(start).Seconds())
        processedCounter.Inc()
        sess.MarkMessage(msg, "")
    }
    return nil
}

func (h *consumerGroupHandler) processMessage(msg *sarama.ConsumerMessage) error {
    // 1) Unmarshal protobuf
    var req ingestpb.LogRequest
    if err := proto.Unmarshal(msg.Value, &req); err != nil {
        return fmt.Errorf("proto unmarshal: %w", err)
    }

    // 2) Parse project UUID
    pid, err := gocql.ParseUUID(req.ProjectId)
    if err != nil {
        return fmt.Errorf("parse project_id: %w", err)
    }

    // 3) Convert timestamp
    ts := time.Unix(0, req.Payload.Timestamp)

    // 4) Write to Cassandra with TTL
    cql := `
      INSERT INTO logs.events
        (project_id, kafka_partition, kafka_offset, event_time, event_name, data)
      VALUES (?, ?, ?, ?, ?, ?)
      USING TTL ?`
    if err := h.cassSess.Query(cql,
        pid,
        msg.Partition,
        msg.Offset,
        ts,
        req.Payload.Name,
        req.Payload.Data,
        h.ttl,
    ).Exec(); err != nil {
        return fmt.Errorf("cassandra insert: %w", err)
    }

    // 5) Write to ClickHouse
    chSQL := `
      INSERT INTO logs
        (project_id, timestamp, event_name, data, kafka_partition, kafka_offset)
      VALUES (?, ?, ?, ?, ?, ?)`
    if _, err := h.chDB.Exec(chSQL,
        req.ProjectId,
        ts,
        req.Payload.Name,
        req.Payload.Data,
        msg.Partition,
        msg.Offset,
    ); err != nil {
        return fmt.Errorf("clickhouse insert: %w", err)
    }

    return nil
}

func main() {
    // Logger
    if err := lib.InitLogger(); err != nil {
        fmt.Fprintf(os.Stderr, "logger init: %v\n", err)
        os.Exit(1)
    }
    logger := lib.Log
    defer logger.Sync()

    // Config
    viper.SetConfigFile("/etc/processor/config.yml")
    if err := lib.LoadConfig(viper.ConfigFileUsed()); err != nil {
        logger.Fatal("config load failed", zap.Error(err))
    }
    var cfg processorConfig
    if err := viper.Unmarshal(&cfg); err != nil {
        logger.Fatal("config unmarshal failed", zap.Error(err))
    }

    // Metrics HTTP endpoint
    initMetrics()
    go func() {
        logger.Info("metrics listening", zap.String("port", cfg.Metrics.Port))
        if err := http.ListenAndServe(":"+cfg.Metrics.Port, nil); err != nil {
            logger.Fatal("metrics server failed", zap.Error(err))
        }
    }()

    // Cassandra session
    cassSess, err := lib.NewCassandraSession(cfg.Cassandra.Hosts)
    if err != nil {
        logger.Fatal("cassandra connect failed", zap.Error(err))
    }
    defer cassSess.Close()

    // ClickHouse connection
    chDB, err := lib.NewClickHouseConn(cfg.ClickHouse.Dsn)
    if err != nil {
        logger.Fatal("clickhouse connect failed", zap.Error(err))
    }
    defer chDB.Close()

    // Kafka consumer group
    consumerGroup, err := lib.NewKafkaConsumer(cfg.Kafka.Brokers, cfg.Kafka.Group, []string{cfg.Kafka.Topic})
    if err != nil {
        logger.Fatal("kafka consumer init failed", zap.Error(err))
    }
    ctx, cancel := context.WithCancel(context.Background())
    handler := &consumerGroupHandler{
        logger:   logger,
        cassSess: cassSess,
        chDB:     chDB,
        ttl:      cfg.Processor.TtlSeconds,
    }
    go func() {
        for {
            if err := consumerGroup.Consume(ctx, []string{cfg.Kafka.Topic}, handler); err != nil {
                logger.Error("consumer error", zap.Error(err))
                time.Sleep(time.Second)
            }
            if ctx.Err() != nil {
                return
            }
        }
    }()

    // Graceful shutdown on SIGINT/SIGTERM
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    <-sigs
    logger.Info("shutting down processor")
    cancel()
}