package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

type Metrics struct {
	pingElapsed                *prometheus.HistogramVec
	DuplicateConnectionCounter prometheus.Counter
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

	metrics.pingElapsed = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "receptor_controller_ping_seconds",
		Help: "Number of seconds spent waiting on a synchronous ping",
	},
		[]string{"account", "recipient"},
	)

	metrics.DuplicateConnectionCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_duplicate_connection_count",
		Help: "The number of receptor websocket connections with the same account number and node id",
	})

	return metrics
}

type DurationRecorder struct {
	elapsed   *prometheus.HistogramVec
	labels    prometheus.Labels
	startTime time.Time
}

func (dr *DurationRecorder) Start() {
	dr.startTime = time.Now()
}

func (dr *DurationRecorder) Stop() {
	recordedDuration := time.Since(dr.startTime)
	if dr.elapsed != nil {
		dr.elapsed.With(dr.labels).Observe(recordedDuration.Seconds())
	}
}

var (
	metrics = NewMetrics()
)
