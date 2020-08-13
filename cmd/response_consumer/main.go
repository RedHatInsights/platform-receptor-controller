package main

import (
	"context"
	"fmt"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"

	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

func main() {
	cfg := config.GetConfig()

	groupID, err := uuid.NewRandom()
	if err != nil {
		fmt.Println("Unable to generate random uuid:", err)
		return
	}

	kafkaConfig := kafka.ReaderConfig{
		Brokers:     cfg.KafkaBrokers,
		Topic:       cfg.KafkaResponsesTopic,
		GroupID:     groupID.String(),
		StartOffset: cfg.KafkaConsumerOffset,
	}

	kafkaReader := kafka.NewReader(kafkaConfig)

	fmt.Println("Kafka consumer config: ", kafkaReader.Config())

	for {
		m, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			fmt.Println("Got an error:", err)
			break
		}
		fmt.Printf("message at offset %d, partition %d: %s = %s\n", m.Offset, m.Partition, string(m.Key), string(m.Value))
	}

	kafkaReader.Close()
}
