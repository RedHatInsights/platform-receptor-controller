package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	ENV_PREFIX = "RECEPTOR_CONTROLLER"

	HANDSHAKE_READ_WAIT            = "WebSocket_Handshake_Read_Wait"
	WRITE_WAIT                     = "WebSocket_Write_Wait"
	PONG_WAIT                      = "WebSocket_Pong_Wait"
	PING_PERIOD                    = "WebSocket_Ping_Period"
	MAX_MESSAGE_SIZE               = "WebSocket_Max_Message_Size"
	SERVICE_TO_SERVICE_CREDENTIALS = "Service_To_Service_Credentials"
	BROKERS                        = "Kafka_Brokers"
	JOBS_TOPIC                     = "Kafka_Jobs_Topic"
	JOBS_GROUP_ID                  = "Kafka_Jobs_Group_Id"
	JOBS_CONSUMER_OFFSET           = "Kafka_Jobs_Consumer_Offset"
	RESPONSES_TOPIC                = "Kafka_Responses_Topic"
	DEFAULT_BROKER_ADDRESS         = "kafka:29092"

	// FIXME: I don't think this belongs here
	NODE_ID = "ReceptorControllerNodeId"
)

type ReceptorControllerConfig struct {
	HandshakeReadWait           time.Duration
	WriteWait                   time.Duration
	PongWait                    time.Duration
	PingPeriod                  time.Duration
	MaxMessageSize              int64
	ServiceToServiceCredentials map[string]interface{}
	ReceptorControllerNodeId    string
	KafkaBrokers                []string
	KafkaJobsTopic              string
	KafkaResponsesTopic         string
	KafkaGroupID                string
	KafkaConsumerOffset         int64
}

func (rcc ReceptorControllerConfig) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", HANDSHAKE_READ_WAIT, rcc.HandshakeReadWait)
	fmt.Fprintf(&b, "%s: %s\n", WRITE_WAIT, rcc.WriteWait)
	fmt.Fprintf(&b, "%s: %s\n", PONG_WAIT, rcc.PongWait)
	fmt.Fprintf(&b, "%s: %s\n", PING_PERIOD, rcc.PingPeriod)
	fmt.Fprintf(&b, "%s: %d\n", MAX_MESSAGE_SIZE, rcc.MaxMessageSize)
	fmt.Fprintf(&b, "%s: %s\n", NODE_ID, rcc.ReceptorControllerNodeId)
	fmt.Fprintf(&b, "%s: %s\n", BROKERS, rcc.KafkaBrokers)
	fmt.Fprintf(&b, "%s: %s\n", JOBS_TOPIC, rcc.KafkaJobsTopic)
	fmt.Fprintf(&b, "%s: %s\n", RESPONSES_TOPIC, rcc.KafkaResponsesTopic)
	fmt.Fprintf(&b, "%s: %s\n", JOBS_GROUP_ID, rcc.KafkaGroupID)
	fmt.Fprintf(&b, "%s: %d", JOBS_CONSUMER_OFFSET, rcc.KafkaConsumerOffset)
	return b.String()
}

func GetConfig() *ReceptorControllerConfig {
	options := viper.New()

	options.SetDefault(HANDSHAKE_READ_WAIT, 5)
	options.SetDefault(WRITE_WAIT, 5)
	options.SetDefault(PONG_WAIT, 25)
	options.SetDefault(MAX_MESSAGE_SIZE, 1*1024*1024)
	options.SetDefault(SERVICE_TO_SERVICE_CREDENTIALS, "")
	options.SetDefault(NODE_ID, "node-cloud-receptor-controller")
	options.SetDefault(BROKERS, []string{DEFAULT_BROKER_ADDRESS})
	options.SetDefault(JOBS_TOPIC, "platform.receptor-controller.jobs")
	options.SetDefault(RESPONSES_TOPIC, "platform.receptor-controller.responses")
	options.SetDefault(JOBS_GROUP_ID, "receptor-controller")
	options.SetDefault(JOBS_CONSUMER_OFFSET, -1)
	options.SetEnvPrefix(ENV_PREFIX)
	options.AutomaticEnv()

	writeWait := options.GetDuration(WRITE_WAIT) * time.Second
	pongWait := options.GetDuration(PONG_WAIT) * time.Second
	pingPeriod := calculatePingPeriod(pongWait)

	return &ReceptorControllerConfig{
		HandshakeReadWait:           options.GetDuration(HANDSHAKE_READ_WAIT) * time.Second,
		WriteWait:                   writeWait,
		PongWait:                    pongWait,
		PingPeriod:                  pingPeriod,
		MaxMessageSize:              options.GetInt64(MAX_MESSAGE_SIZE),
		ServiceToServiceCredentials: options.GetStringMap(SERVICE_TO_SERVICE_CREDENTIALS),
		ReceptorControllerNodeId:    options.GetString(NODE_ID),
		KafkaBrokers:                options.GetStringSlice(BROKERS),
		KafkaJobsTopic:              options.GetString(JOBS_TOPIC),
		KafkaResponsesTopic:         options.GetString(RESPONSES_TOPIC),
		KafkaGroupID:                options.GetString(JOBS_GROUP_ID),
		KafkaConsumerOffset:         options.GetInt64(JOBS_CONSUMER_OFFSET),
	}
}

func calculatePingPeriod(pongWait time.Duration) time.Duration {
	pingPeriod := (pongWait * 9) / 10
	return pingPeriod
}
