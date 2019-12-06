package queue

import (
	"log"

	kafka "github.com/segmentio/kafka-go"
)

func InitProducer(cfg *KafkaConfig) *kafka.Writer {
	log.Println("Initializing new Kafka producer..")

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
	})

	log.Println("Producing messages to topic: ", cfg.Topic)

	return w
}
