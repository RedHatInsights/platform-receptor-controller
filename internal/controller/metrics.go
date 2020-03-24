package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	//"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	TotalMessagesSentCounter     prometheus.Counter
	TotalMessagesReceivedCounter prometheus.Counter
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

	/*
		metrics.TotalMessagesSentCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "receptor_controller_total_messages_sent_count",
			Help: "The total number of messages sent",
		})

		metrics.TotalMessagesReceivedCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "receptor_controller_total_messages_received_count",
			Help: "The total number of messages received",
		})
	*/

	return metrics
}

var (
	metrics = NewMetrics()
)
