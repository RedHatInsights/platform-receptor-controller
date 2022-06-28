package queue

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	kafka "github.com/segmentio/kafka-go"
)

func StartConsumer(cfg *ConsumerConfig) *kafka.Reader {
	logger.Log.Info("Starting a new kafka consumer...")
	logger.Log.Info("Kafka consumer configuration: ", cfg)

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

	readerConfig := kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     cfg.GroupID,
		StartOffset: cfg.ConsumerOffset,
	}

	if kafkaDialer != nil {
		readerConfig.Dialer = kafkaDialer
	}

	r := kafka.NewReader(readerConfig)

	logger.Log.Info("Kafka consumer config: ", r.Config())

	return r
}
