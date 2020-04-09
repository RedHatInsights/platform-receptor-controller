package protocol

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	payloadMessageSize prometheus.Histogram
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

	metrics.payloadMessageSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "receptor_controller_payload_message_sizes",
		Help: "Size of payloads posted",
		Buckets: []float64{
			1024 * 10,
			1024 * 100,
			1024 * 1000,
			1024 * 10000,
		}})

	return metrics
}

var (
	metrics = NewMetrics()
)
