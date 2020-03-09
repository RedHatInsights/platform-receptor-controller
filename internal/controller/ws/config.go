package ws

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	ENV_PREFIX = "RECEPTOR_CONTROLLER"

	HANDSHAKE_READ_WAIT = "WebSocket_Handshake_Read_Wait"
	WRITE_WAIT          = "WebSocket_Write_Wait"
	PONG_WAIT           = "WebSocket_Pong_Wait"
	PING_PERIOD         = "WebSocket_Ping_Period"
	MAX_MESSAGE_SIZE    = "WebSocket_Max_Message_Size"
	KNOWN_SECRETS       = "Known_Secrets"

	// FIXME: I don't think this belongs here
	NODE_ID = "ReceptorControllerNodeId"
)

type WebSocketConfig struct {
	HandshakeReadWait        time.Duration
	WriteWait                time.Duration
	PongWait                 time.Duration
	PingPeriod               time.Duration
	MaxMessageSize           int64
	KnownSecrets             map[string]interface{}
	ReceptorControllerNodeId string
}

func (wsc WebSocketConfig) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", HANDSHAKE_READ_WAIT, wsc.HandshakeReadWait)
	fmt.Fprintf(&b, "%s: %s\n", WRITE_WAIT, wsc.WriteWait)
	fmt.Fprintf(&b, "%s: %s\n", PONG_WAIT, wsc.PongWait)
	fmt.Fprintf(&b, "%s: %s\n", PING_PERIOD, wsc.PingPeriod)
	fmt.Fprintf(&b, "%s: %d\n", MAX_MESSAGE_SIZE, wsc.MaxMessageSize)
	fmt.Fprintf(&b, "%s: %s", NODE_ID, wsc.ReceptorControllerNodeId)
	return b.String()
}

func GetWebSocketConfig() *WebSocketConfig {
	options := viper.New()

	options.SetDefault(HANDSHAKE_READ_WAIT, 5)
	options.SetDefault(WRITE_WAIT, 5)
	options.SetDefault(PONG_WAIT, 25)
	options.SetDefault(MAX_MESSAGE_SIZE, 1*1024*1024)
	options.SetDefault(KNOWN_SECRETS, "")
	options.SetDefault(NODE_ID, "node-cloud-receptor-controller")
	options.SetEnvPrefix(ENV_PREFIX)
	options.AutomaticEnv()

	writeWait := options.GetDuration(WRITE_WAIT) * time.Second
	pongWait := options.GetDuration(PONG_WAIT) * time.Second
	pingPeriod := calculatePingPeriod(pongWait)

	return &WebSocketConfig{
		HandshakeReadWait:        options.GetDuration(HANDSHAKE_READ_WAIT) * time.Second,
		WriteWait:                writeWait,
		PongWait:                 pongWait,
		PingPeriod:               pingPeriod,
		MaxMessageSize:           options.GetInt64(MAX_MESSAGE_SIZE),
		KnownSecrets:             options.GetStringMap(KNOWN_SECRETS),
		ReceptorControllerNodeId: options.GetString(NODE_ID),
	}
}

func calculatePingPeriod(pongWait time.Duration) time.Duration {
	pingPeriod := (pongWait * 9) / 10
	return pingPeriod
}
