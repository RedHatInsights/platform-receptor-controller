package queue

import (
	"fmt"

	kafka "github.com/segmentio/kafka-go"
)

func InitConsumer(cfg *KafkaConfig) *kafka.Reader {
	fmt.Println("Initializing new kafka consumer...")

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     cfg.GroupID,
		StartOffset: cfg.ConsumerOffset,
	})

	fmt.Println("Kafka consumer config: ", r.Config())

	return r
}
