CREATE TABLE IF NOT EXISTS logs (
  project_id String,
  timestamp DateTime64(9, 'UTC'),
  event_name String,
  data Map(String, String),
  kafka_partition UInt32,
  kafka_offset UInt64
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (project_id, timestamp, kafka_partition, kafka_offset);