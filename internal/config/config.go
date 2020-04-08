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
	SEND_MESSAGE_TIMEOUT           = "WebSocket_Send_Message_Context_Timeout"
	MAX_MESSAGE_SIZE               = "WebSocket_Max_Message_Size"
	SOCKET_BUFFER_SIZE             = "WebSocket_Buffer_Size"
	CHANNEL_BUFFER_SIZE            = "Goroutine_Channel_Buffer_Size"
	SERVICE_TO_SERVICE_CREDENTIALS = "Service_To_Service_Credentials"
	BROKERS                        = "Kafka_Brokers"
	JOBS_TOPIC                     = "Kafka_Jobs_Topic"
	JOBS_GROUP_ID                  = "Kafka_Jobs_Group_Id"
	JOBS_CONSUMER_OFFSET           = "Kafka_Jobs_Consumer_Offset"
	RESPONSES_TOPIC                = "Kafka_Responses_Topic"
	RESPONSES_BATCH_SIZE           = "Kafka_Responses_Batch_Size"
	RESPONSES_BATCH_BYTES          = "Kafka_Responses_Batch_Bytes"
	DEFAULT_BROKER_ADDRESS         = "kafka:29092"

	NODE_ID = "ReceptorControllerNodeId"
)

type Config struct {
	HandshakeReadWait           time.Duration
	WriteWait                   time.Duration
	PongWait                    time.Duration
	PingPeriod                  time.Duration
	SendMessageTimeout          time.Duration
	MaxMessageSize              int64
	SocketBufferSize            int
	ChannelBufferSize           int
	ServiceToServiceCredentials map[string]interface{}
	ReceptorControllerNodeId    string
	KafkaBrokers                []string
	KafkaJobsTopic              string
	KafkaResponsesTopic         string
	KafkaResponsesBatchSize     int
	KafkaResponsesBatchBytes    int
	KafkaGroupID                string
	KafkaConsumerOffset         int64
}

func (c Config) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", HANDSHAKE_READ_WAIT, c.HandshakeReadWait)
	fmt.Fprintf(&b, "%s: %s\n", WRITE_WAIT, c.WriteWait)
	fmt.Fprintf(&b, "%s: %s\n", PONG_WAIT, c.PongWait)
	fmt.Fprintf(&b, "%s: %s\n", PING_PERIOD, c.PingPeriod)
	fmt.Fprintf(&b, "%s: %s\n", SEND_MESSAGE_TIMEOUT, c.SendMessageTimeout)
	fmt.Fprintf(&b, "%s: %d\n", MAX_MESSAGE_SIZE, c.MaxMessageSize)
	fmt.Fprintf(&b, "%s: %d\n", SOCKET_BUFFER_SIZE, c.SocketBufferSize)
	fmt.Fprintf(&b, "%s: %d\n", CHANNEL_BUFFER_SIZE, c.ChannelBufferSize)
	fmt.Fprintf(&b, "%s: %s\n", NODE_ID, c.ReceptorControllerNodeId)
	fmt.Fprintf(&b, "%s: %s\n", BROKERS, c.KafkaBrokers)
	fmt.Fprintf(&b, "%s: %s\n", JOBS_TOPIC, c.KafkaJobsTopic)
	fmt.Fprintf(&b, "%s: %s\n", RESPONSES_TOPIC, c.KafkaResponsesTopic)
	fmt.Fprintf(&b, "%s: %d\n", RESPONSES_BATCH_SIZE, c.KafkaResponsesBatchSize)
	fmt.Fprintf(&b, "%s: %d\n", RESPONSES_BATCH_BYTES, c.KafkaResponsesBatchBytes)
	fmt.Fprintf(&b, "%s: %s\n", JOBS_GROUP_ID, c.KafkaGroupID)
	fmt.Fprintf(&b, "%s: %d", JOBS_CONSUMER_OFFSET, c.KafkaConsumerOffset)
	return b.String()
}

func GetConfig() *Config {
	options := viper.New()

	options.SetDefault(HANDSHAKE_READ_WAIT, 5)
	options.SetDefault(WRITE_WAIT, 5)
	options.SetDefault(PONG_WAIT, 25)
	options.SetDefault(SEND_MESSAGE_TIMEOUT, 10)
	options.SetDefault(MAX_MESSAGE_SIZE, 1*1024*1024)
	options.SetDefault(SOCKET_BUFFER_SIZE, 1024)
	options.SetDefault(CHANNEL_BUFFER_SIZE, 10)
	options.SetDefault(SERVICE_TO_SERVICE_CREDENTIALS, "")
	options.SetDefault(NODE_ID, "node-cloud-receptor-controller")
	options.SetDefault(BROKERS, []string{DEFAULT_BROKER_ADDRESS})
	options.SetDefault(JOBS_TOPIC, "platform.receptor-controller.jobs")
	options.SetDefault(RESPONSES_TOPIC, "platform.receptor-controller.responses")
	options.SetDefault(RESPONSES_BATCH_SIZE, 100)
	options.SetDefault(RESPONSES_BATCH_BYTES, 1048576)
	options.SetDefault(JOBS_GROUP_ID, "receptor-controller")
	options.SetDefault(JOBS_CONSUMER_OFFSET, -1)
	options.SetEnvPrefix(ENV_PREFIX)
	options.AutomaticEnv()

	writeWait := options.GetDuration(WRITE_WAIT) * time.Second
	pongWait := options.GetDuration(PONG_WAIT) * time.Second
	pingPeriod := calculatePingPeriod(pongWait)

	return &Config{
		HandshakeReadWait:           options.GetDuration(HANDSHAKE_READ_WAIT) * time.Second,
		WriteWait:                   writeWait,
		PongWait:                    pongWait,
		PingPeriod:                  pingPeriod,
		SendMessageTimeout:          options.GetDuration(SEND_MESSAGE_TIMEOUT) * time.Second,
		MaxMessageSize:              options.GetInt64(MAX_MESSAGE_SIZE),
		SocketBufferSize:            options.GetInt(SOCKET_BUFFER_SIZE),
		ChannelBufferSize:           options.GetInt(CHANNEL_BUFFER_SIZE),
		ServiceToServiceCredentials: options.GetStringMap(SERVICE_TO_SERVICE_CREDENTIALS),
		ReceptorControllerNodeId:    options.GetString(NODE_ID),
		KafkaBrokers:                options.GetStringSlice(BROKERS),
		KafkaJobsTopic:              options.GetString(JOBS_TOPIC),
		KafkaResponsesTopic:         options.GetString(RESPONSES_TOPIC),
		KafkaResponsesBatchSize:     options.GetInt(RESPONSES_BATCH_SIZE),
		KafkaResponsesBatchBytes:    options.GetInt(RESPONSES_BATCH_BYTES),
		KafkaGroupID:                options.GetString(JOBS_GROUP_ID),
		KafkaConsumerOffset:         options.GetInt64(JOBS_CONSUMER_OFFSET),
	}
}

func calculatePingPeriod(pongWait time.Duration) time.Duration {
	pingPeriod := (pongWait * 9) / 10
	return pingPeriod
}
