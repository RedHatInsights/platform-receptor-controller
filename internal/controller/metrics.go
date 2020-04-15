package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

type Metrics struct {
	pingElapsed                          *prometheus.HistogramVec
	duplicateConnectionCounter           prometheus.Counter
	responseKafkaWriterGoRoutineGauge    prometheus.Gauge
	responseKafkaWriterFailureCounter    prometheus.Counter
	responseMessageWithoutHandlerCounter prometheus.Counter
	responseMessageHandledCounter        prometheus.Counter
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

	metrics.pingElapsed = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "receptor_controller_ping_seconds",
		Help: "Number of seconds spent waiting on a synchronous ping",
	},
		[]string{"account", "recipient"},
	)

	metrics.duplicateConnectionCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_duplicate_connection_count",
		Help: "The number of receptor websocket connections with the same account number and node id",
	})

	metrics.responseKafkaWriterGoRoutineGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "receptor_controller_kafka_response_writer_go_routine_count",
		Help: "The total number of active kakfa response writer go routines",
	})

	metrics.responseKafkaWriterFailureCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_kafka_response_writer_failure_count",
		Help: "The number of responses that failed to get produced to kafka topic",
	})

	metrics.responseMessageWithoutHandlerCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_response_message_without_handler_count",
		Help: "The number of response messages received that do not have a handler",
	})

	metrics.responseMessageHandledCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_response_message_handled_count",
		Help: "The number of response messages handled",
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
