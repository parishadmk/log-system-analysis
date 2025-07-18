-- Create a keyspace for raw logs
CREATE KEYSPACE IF NOT EXISTS logs WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': '1'
};

-- Table for raw events, TTL set per‐insert
CREATE TABLE IF NOT EXISTS logs.events (
  project_id UUID,
  kafka_partition int,
  kafka_offset bigint,
  event_time timestamp,
  event_name text,
  data map<text, text>,
  PRIMARY KEY ((project_id), kafka_partition, kafka_offset)
);