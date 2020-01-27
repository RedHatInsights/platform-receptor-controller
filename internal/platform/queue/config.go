package queue

import (
	"github.com/spf13/viper"
)

type ConsumerConfig struct {
	Brokers        []string
	Topic          string
	GroupID        string
	ConsumerOffset int64
}

type ProducerConfig struct {
	Brokers []string
	Topic   string
}

func GetConsumer() *ConsumerConfig {
	options := viper.New()
	options.SetDefault("Brokers", []string{"kafka:29092"})
	options.SetDefault("Topic", "platform.receptor-controller.jobs")
	options.SetDefault("GroupID", "receptor-controller")
	options.SetDefault("ConsumerOffset", -1)

	return &ConsumerConfig{
		Brokers:        options.GetStringSlice("Brokers"),
		Topic:          options.GetString("Topic"),
		GroupID:        options.GetString("GroupID"),
		ConsumerOffset: options.GetInt64("ConsumerOffset"),
	}
}

func GetProducer() *ProducerConfig {
	options := viper.New()
	options.SetDefault("Brokers", []string{"kafka:29092"})
	options.SetDefault("Topic", "platform.receptor-controller.responses")

	return &ProducerConfig{
		Brokers: options.GetStringSlice("Brokers"),
		Topic:   options.GetString("Topic"),
	}
}
