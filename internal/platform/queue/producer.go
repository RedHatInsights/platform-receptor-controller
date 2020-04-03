package queue

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	kafka "github.com/segmentio/kafka-go"
)

func StartProducer(cfg *ProducerConfig) *kafka.Writer {
	logger.Log.Info("Starting a new Kafka producer..")
	logger.Log.Info("Kafka producer configuration: ", cfg)

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: cfg.Brokers,
		Topic:   cfg.Topic,
	})

	logger.Log.Info("Producing messages to topic: ", cfg.Topic)

	return w
}
