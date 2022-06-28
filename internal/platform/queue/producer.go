package queue

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	kafka "github.com/segmentio/kafka-go"
)

func StartProducer(cfg *ProducerConfig) *kafka.Writer {
	logger.Log.Info("Starting a new Kafka producer..")
	logger.Log.Info("Kafka producer configuration: ", cfg)

	var kafkaDialer *kafka.Dialer
	var err error

	globalConfig := config.GetConfig()

	if globalConfig.KafkaSaslUsername != "" {

		kafkaDialer, err = saslDialer(&SaslConfig{
			SaslMechanism: globalConfig.KafkaSaslMechanism,
			SaslUsername:  globalConfig.KafkaSaslUsername,
			SaslPassword:  globalConfig.KafkaSaslPassword,
			KafkaCA:       globalConfig.KafkaCAPath,
		})
		if err != nil {
			logger.Log.Error("Failed to create a new Kafka Dialer: ", err)
			panic(err)
		}
	}

	writerConfig := kafka.WriterConfig{
		Brokers:    cfg.Brokers,
		Topic:      cfg.Topic,
		BatchSize:  cfg.BatchSize,
		BatchBytes: cfg.BatchBytes,
	}

	if kafkaDialer != nil {
		writerConfig.Dialer = kafkaDialer
	}

	w := kafka.NewWriter(writerConfig)

	logger.Log.Info("Producing messages to topic: ", cfg.Topic)

	return w
}
