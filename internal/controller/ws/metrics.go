package ws

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	TotalConnectionCounter       prometheus.Counter
	ActiveConnectionCounter      prometheus.Gauge
	ErrorConnectionCounter       prometheus.Counter
	TotalMessagesSentCounter     prometheus.Counter
	TotalMessagesReceivedCounter prometheus.Counter
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

	metrics.TotalConnectionCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_total_websocket_connection_count",
		Help: "The total number of receptor websocket connections received",
	})

	metrics.ActiveConnectionCounter = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "receptor_controller_active_websocket_connection_count",
		Help: "The total number of active receptor websocket connections",
	})

	/*
		metrics.ErrorConnectionCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "receptor_controller_error_connection_count",
			Help: "The total number of receptor websocket connection errors",
		})
	*/

	metrics.TotalMessagesSentCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_total_messages_sent_count",
		Help: "The total number of messages sent",
	})

	metrics.TotalMessagesReceivedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_total_messages_received_count",
		Help: "The total number of messages received",
	})

	return metrics
}

var (
	metrics = NewMetrics()
)
