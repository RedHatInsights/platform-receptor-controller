package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

type Metrics struct {
	pingElapsed *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

	metrics.pingElapsed = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "receptor_controller_ping_seconds",
		Help: "Number of seconds spent waiting on a synchronous ping",
	},
		[]string{"account", "recipient"},
	)

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

type DurationRecorder struct {
	elapsed   *prometheus.HistogramVec
	labels    prometheus.Labels
	startTime time.Time
}

func (dr *DurationRecorder) Start(elapsed *prometheus.HistogramVec, labels prometheus.Labels) {
	dr.elapsed = elapsed
	dr.labels = labels
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
