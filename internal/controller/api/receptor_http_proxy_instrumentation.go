package api

import (
	"context"
	"strconv"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/middlewares"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	metrics = newMetrics()
)

type receptorHttpProxyProbe struct {
	logger        *logrus.Entry
	operationName string
}

func createProbe(ctx context.Context, operationName string) *receptorHttpProxyProbe {

	principal, _ := middlewares.GetPrincipal(ctx)
	requestId := request_id.GetReqID(ctx)

	logger := logger.Log.WithFields(logrus.Fields{"account": principal.GetAccount(),
		"request_id": requestId})

	return &receptorHttpProxyProbe{logger: logger, operationName: operationName}
}

func (rhpp *receptorHttpProxyProbe) failedToMarshalPayload(err error) {
	metrics.receptorProxyRemoteCallFailureCounter.With(
		prometheus.Labels{"operation": rhpp.operationName}).Inc()
	logError(rhpp.logger, err, rhpp.operationName+" - Failed to marshal JSON payload.")
}

func (rhpp *receptorHttpProxyProbe) failedToMakeHttpRequest(err error) {
	metrics.receptorProxyRemoteCallFailureCounter.With(
		prometheus.Labels{"operation": rhpp.operationName}).Inc()
	logError(rhpp.logger, err, rhpp.operationName+" - Failed to create HTTP Request.")
}

func (rhpp *receptorHttpProxyProbe) recordHttpStatusCode(statusCode int) {
	metrics.receptorProxyRemoteCallStatusCodeCounter.With(
		prometheus.Labels{"operation": rhpp.operationName,
			"status_code": strconv.Itoa(statusCode)}).Inc()
}

func (rhpp *receptorHttpProxyProbe) invalidHttpStatusCode(statusCode int) {
	strStatusCode := strconv.Itoa(statusCode)
	metrics.receptorProxyRemoteCallStatusCodeCounter.With(
		prometheus.Labels{"operation": rhpp.operationName,
			"status_code": strStatusCode}).Inc()
	rhpp.logger.WithFields(logrus.Fields{"status_code": statusCode}).Error(rhpp.operationName + " - Invalid HTTP status code - " + strStatusCode)
}

func (rhpp *receptorHttpProxyProbe) failedToUnmarshalResponse(err error) {
	metrics.receptorProxyRemoteCallFailureCounter.With(
		prometheus.Labels{"operation": rhpp.operationName}).Inc()
	logError(rhpp.logger, err, rhpp.operationName+" - Unable to parse json response from receptor-gateway.")
}

func (rhpp *receptorHttpProxyProbe) sendingMessage(accountNumber, recipient string) {
	rhpp.logger.Infof("Sending message to receptor-gateway - %s:%s\n", accountNumber, recipient)
}

func (rhpp *receptorHttpProxyProbe) messageSent(messageID uuid.UUID) {
	metrics.receptorProxyRemoteCallCounter.With(
		prometheus.Labels{"operation": "message"}).Inc()
	rhpp.logger.WithFields(logrus.Fields{"message_id": messageID}).Info("Message sent to receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) sendingPing(accountNumber, recipient string) {
	rhpp.logger.Infof("Sending ping message to receptor-gateway - %s:%s\n", accountNumber, recipient)
}

func (rhpp *receptorHttpProxyProbe) pingMessageSent() {
	metrics.receptorProxyRemoteCallCounter.With(
		prometheus.Labels{"operation": "ping"}).Inc()
	rhpp.logger.Info("Ping message sent to receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) closingConnection(accountNumber, recipient string) {
	rhpp.logger.WithFields(logrus.Fields{"recipient": recipient}).Info("Sending close message to receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) connectionClosed(accountNumber, recipient string) {
	metrics.receptorProxyRemoteCallCounter.With(
		prometheus.Labels{"operation": "close_connection"}).Inc()
	rhpp.logger.WithFields(logrus.Fields{"recipient": recipient}).Info("Sent close message to receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) gettingCapabilities(accountNumber, recipient string) {
	rhpp.logger.WithFields(logrus.Fields{"recipient": recipient}).Info("Getting node capabilities from receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) retrievedCapabilities(accountNumber, recipient string) {
	metrics.receptorProxyRemoteCallCounter.With(
		prometheus.Labels{"operation": "get_capabilities"}).Inc()
	rhpp.logger.WithFields(logrus.Fields{"recipient": recipient}).Info("Got node capabilities from receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) recordRemoteCallDuration(callDuration time.Duration) {
	metrics.receptorProxyRemoteCallDuration.With(
		prometheus.Labels{"operation": rhpp.operationName}).Observe(callDuration.Seconds())
}

func logError(logger *logrus.Entry, err error, errMsg string) {
	logger.WithFields(logrus.Fields{"error": err}).Error(errMsg)
}

type receptorHttpProxyMetrics struct {
	receptorProxyRemoteCallCounter           *prometheus.CounterVec
	receptorProxyRemoteCallFailureCounter    *prometheus.CounterVec
	receptorProxyRemoteCallStatusCodeCounter *prometheus.CounterVec
	receptorProxyRemoteCallDuration          *prometheus.SummaryVec
}

func newMetrics() *receptorHttpProxyMetrics {
	metrics := new(receptorHttpProxyMetrics)

	metrics.receptorProxyRemoteCallCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_remote_call_count",
		Help: "The number of receptor messages sent from the job-receiver to the gateway",
	}, []string{"operation"})

	metrics.receptorProxyRemoteCallFailureCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_remote_call_failure_count",
		Help: "The number of receptor messages that failed to be sent from the job-receiver to the gateway",
	}, []string{"operation"})

	metrics.receptorProxyRemoteCallStatusCodeCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_remote_operation_status_code_count",
		Help: "Count of response codes received by the receptor http proxy",
	}, []string{"operation", "status_code"})

	metrics.receptorProxyRemoteCallDuration = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name: "receptor_controller_receptor_proxy_remote_operation_duration",
		Help: "Number of seconds spent waiting on receptor-gateway",
	}, []string{"operation"})

	return metrics
}
