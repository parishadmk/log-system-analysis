package lib

import (
    "github.com/IBM/sarama"
)

func NewKafkaProducer(brokers []string) (sarama.SyncProducer, error) {
    cfg := sarama.NewConfig()
    cfg.Producer.Return.Successes = true
    return sarama.NewSyncProducer(brokers, cfg)
}

func NewKafkaConsumer(brokers []string, groupID string, topics []string) (sarama.ConsumerGroup, error) {
    cfg := sarama.NewConfig()
    cfg.Version = sarama.V2_8_0_0
    return sarama.NewConsumerGroup(brokers, groupID, cfg)
}