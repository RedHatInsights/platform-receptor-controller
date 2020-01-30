package queue

import (
	"log"

	kafka "github.com/segmentio/kafka-go"
)

func StartConsumer(cfg *ConsumerConfig) *kafka.Reader {
	log.Println("Starting a new kafka consumer...")

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     cfg.GroupID,
		StartOffset: cfg.ConsumerOffset,
	})

	log.Println("Kafka consumer config: ", r.Config())

	return r
}
