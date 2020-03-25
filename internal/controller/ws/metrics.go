package ws

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	TotalConnectionCounter       prometheus.Counter
	ActiveConnectionCounter      prometheus.Gauge
	TotalMessagesSentCounter     prometheus.Counter
	TotalMessagesReceivedCounter prometheus.Counter
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

	metrics.TotalConnectionCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_websocket_total_connection_count",
		Help: "The total number of receptor websocket connections received",
	})

	metrics.ActiveConnectionCounter = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "receptor_controller_websocket_active_connection_count",
		Help: "The total number of active receptor websocket connections",
	})

	metrics.TotalMessagesSentCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_websocket_total_messages_sent_count",
		Help: "The total number of messages sent over a websocket connection",
	})

	metrics.TotalMessagesReceivedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_websocket_total_messages_received_count",
		Help: "The total number of messages received over a websocket connection",
	})

	return metrics
}

var (
	metrics = NewMetrics()
)
