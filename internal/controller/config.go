package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type WebSocketConfig struct {
	WriteWait                time.Duration
	PongWait                 time.Duration
	PingPeriod               time.Duration
	MaxMessageSize           int
	ReceptorControllerNodeId string
}

func (wsc WebSocketConfig) String() string {
	var b strings.Builder
	componentName := "WebSocket"
	fmt.Fprintf(&b, "%s WriteWait: %s\n", componentName, wsc.WriteWait)
	fmt.Fprintf(&b, "%s PongWait: %s\n", componentName, wsc.PongWait)
	fmt.Fprintf(&b, "%s PingPeriod: %s\n", componentName, wsc.PingPeriod)
	fmt.Fprintf(&b, "%s MaxMessageSize: %d\n", componentName, wsc.MaxMessageSize)
	fmt.Fprintf(&b, "%s ReceptorControllerNodeID: %s\n", componentName, wsc.ReceptorControllerNodeId)
	return b.String()
}

func Get() *WebSocketConfig {
	options := viper.New()

	options.SetDefault("WriteWait", 5)
	options.SetDefault("PongWait", 25)
	options.SetDefault("MaxMessageSize", 1024*1024)
	options.SetDefault("ReceptorControllerNodeId", "node-cloud-receptor-controller")

	writeWait := options.GetDuration("WriteWait") * time.Second
	pongWait := options.GetDuration("PongWait") * time.Second
	pingPeriod := calculatePingPeriod(pongWait)

	return &WebSocketConfig{
		WriteWait:                writeWait,
		PongWait:                 pongWait,
		PingPeriod:               pingPeriod,
		MaxMessageSize:           options.GetInt("MaxMessageSize"),
		ReceptorControllerNodeId: options.GetString("ReceptorControllerNodeId"),
	}
}

func calculatePingPeriod(pongWait time.Duration) time.Duration {
	pingPeriod := (pongWait * 9) / 10
	return pingPeriod
}
