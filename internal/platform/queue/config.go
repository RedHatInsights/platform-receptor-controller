package queue

import (
	"github.com/spf13/viper"
)

type KafkaConfig struct {
	Brokers        []string
	Topic          string
	GroupID        string
	ConsumerOffset int64
}

func Get() *KafkaConfig {
	options := viper.New()
	options.SetDefault("Brokers", []string{"kafka:29092"})
	options.SetDefault("Topic", "platform.receptor-controller-in")
	options.SetDefault("GroupID", "receptor-controller")
	options.SetDefault("ConsumerOffset", -1)

	return &KafkaConfig{
		Brokers:        options.GetStringSlice("Brokers"),
		Topic:          options.GetString("Topic"),
		GroupID:        options.GetString("GroupID"),
		ConsumerOffset: options.GetInt64("ConsumerOffset"),
	}
}
