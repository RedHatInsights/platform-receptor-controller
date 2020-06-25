package api

import (
	"context"

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
	logger *logrus.Entry
}

func createProbe(ctx context.Context) *receptorHttpProxyProbe {

	principal, _ := middlewares.GetPrincipal(ctx)
	requestId := request_id.GetReqID(ctx)

	logger := logger.Log.WithFields(logrus.Fields{"account": principal.GetAccount(),
		"request_id": requestId})

	return &receptorHttpProxyProbe{logger: logger}
}

func (rhpp *receptorHttpProxyProbe) sendingMessage(accountNumber, recipient string) {
	rhpp.logger.Infof("Sending message to receptor-gateway - %s:%s\n", accountNumber, recipient)
}

func (rhpp *receptorHttpProxyProbe) messageSent(messageID uuid.UUID) {
	metrics.receptorProxyMessageSentCounter.Inc()
	rhpp.logger.WithFields(logrus.Fields{"message_id": messageID}).Info("Message sent to receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) failedToSendMessage(errorMsg string, err error) {
	metrics.receptorProxyMessageSendFailureCounter.Inc()
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *receptorHttpProxyProbe) sendingPing(accountNumber, recipient string) {
	rhpp.logger.Infof("Sending ping message to receptor-gateway - %s:%s\n", accountNumber, recipient)
}

func (rhpp *receptorHttpProxyProbe) pingMessageSent() {
	metrics.receptorProxyPingMessageSentCounter.Inc()
	rhpp.logger.Info("Ping message sent to receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) closingConnection(accountNumber, recipient string) {
	rhpp.logger.Info("Sending close message to receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) gettingCapabilities(accountNumber, recipient string) {
	rhpp.logger.Info("Getting node capabilities from receptor-gateway")
}

func (rhpp *receptorHttpProxyProbe) failedToProcessMessageResponse(errorMsg string, err error) {
	metrics.receptorProxyMessageResponseProcessFailureCounter.Inc()
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *receptorHttpProxyProbe) failedToSendPingMessage(errorMsg string, err error) {
	metrics.receptorProxyPingMessageSendFailureCounter.Inc()
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *receptorHttpProxyProbe) failedToProcessPingMessageResponse(errorMsg string, err error) {
	metrics.receptorProxyPingMessageResponseProcessFailureCounter.Inc()
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *receptorHttpProxyProbe) failedToSendCloseConnectionMessage(errorMsg string, err error) {
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *receptorHttpProxyProbe) failedToRetrieveCapabilities(errorMsg string, err error) {
	logError(rhpp.logger, err, errorMsg)
}

func logError(logger *logrus.Entry, err error, errMsg string) {
	logger.WithFields(logrus.Fields{"error": err}).Error(errMsg)
}

type receptorHttpProxyMetrics struct {
	receptorProxyMessageSentCounter                   prometheus.Counter
	receptorProxyMessageSendFailureCounter            prometheus.Counter
	receptorProxyMessageResponseProcessFailureCounter prometheus.Counter

	receptorProxyPingMessageSentCounter                   prometheus.Counter
	receptorProxyPingMessageSendFailureCounter            prometheus.Counter
	receptorProxyPingMessageResponseProcessFailureCounter prometheus.Counter
}

func newMetrics() *receptorHttpProxyMetrics {
	metrics := new(receptorHttpProxyMetrics)

	metrics.receptorProxyMessageSentCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_messages_sent_count",
		Help: "The number of receptor messages sent from the job-receiver to the gateway",
	})

	metrics.receptorProxyMessageSendFailureCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_messages_send_failure_count",
		Help: "The number of receptor messages that failed to be sent from the job-receiver to the gateway",
	})

	metrics.receptorProxyMessageResponseProcessFailureCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_message_response_process_failure_count",
		Help: "The number of receptor message responses that failed to be processed from the gateway",
	})

	metrics.receptorProxyPingMessageSentCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_ping_message_sent_count",
		Help: "The number of receptor ping messages sent from the job-receiver to the gateway",
	})

	metrics.receptorProxyPingMessageSendFailureCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_ping_messages_send_failure_count",
		Help: "The number of receptor ping messages that failed to be sent from the job-receiver to the gateway",
	})

	metrics.receptorProxyPingMessageResponseProcessFailureCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "receptor_controller_receptor_proxy_ping_message_response_process_failure_count",
		Help: "The number of receptor message ping responses that failed to be processed from the gateway",
	})

	return metrics
}
