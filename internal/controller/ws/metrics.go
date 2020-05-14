package ws

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	TotalConnectionCounter       *prometheus.CounterVec
	ActiveConnectionCounter      *prometheus.GaugeVec
	TotalMessagesSentCounter     prometheus.Counter
	TotalMessagesReceivedCounter prometheus.Counter
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

	metrics.TotalConnectionCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "receptor_controller_websocket_total_connection_count",
		Help: "The total number of receptor websocket connections received",
	},
		[]string{"account"},
	)

	metrics.ActiveConnectionCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "receptor_controller_websocket_active_connection_count",
		Help: "The total number of active receptor websocket connections",
	},
		[]string{"account"},
	)

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
