package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	duplicateConnectionCounter           prometheus.Counter
	responseKafkaWriterGoRoutineGauge    prometheus.Gauge
	responseKafkaWriterFailureCounter    prometheus.Counter
	responseMessageWithoutHandlerCounter prometheus.Counter
	responseMessageHandledCounter        prometheus.Counter
	messageDirectiveCounter              *prometheus.CounterVec

	podRunningStatusLookupFailure                 prometheus.Counter
	autoConnectionClosureDueToDuplicateConnection prometheus.Counter
	reRegisterConnectionWithRedis                 prometheus.Counter
	unregisterStaleConnectionFromRedis            prometheus.Counter
	redisConnectionError                          prometheus.Counter
}

func NewMetrics() *Metrics {
	metrics := new(Metrics)

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

	metrics.podRunningStatusLookupFailure = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_pod_running_status_lookup_failure_count",
		Help: "The number of times a pod running status lookup has failed",
	})

	metrics.autoConnectionClosureDueToDuplicateConnection = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_auto_connection_closure_due_to_duplicate_connection_count",
		Help: "The number of times a connection has been closed due to locating a duplication connection",
	})

	metrics.reRegisterConnectionWithRedis = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_reregister_connection_with_redis_count",
		Help: "The number of times a connection has been re-registered with redis",
	})

	metrics.unregisterStaleConnectionFromRedis = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_unregister_stale_connection_from_redis_count",
		Help: "The number of times a stale connection has been unregistered from redis",
	})

	metrics.redisConnectionError = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_redis_connection_error_count",
		Help: "The number of times a redis connection error has occurred",
	})

	metrics.messageDirectiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "receptor_controller_message_directive_count",
		Help: "The number of messages recieved by the receptor controller per directive",
	}, []string{"directive"})

	return metrics
}

var (
	metrics = NewMetrics()
)
