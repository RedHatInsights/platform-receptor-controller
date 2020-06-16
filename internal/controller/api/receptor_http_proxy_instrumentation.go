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

type ReceptorHttpProxyProbe struct {
	logger *logrus.Entry

	metrics *metrics
}

func createProbe(ctx context.Context) *ReceptorHttpProxyProbe {

	principal, _ := middlewares.GetPrincipal(ctx)
	requestId := request_id.GetReqID(ctx)

	logger := logger.Log.WithFields(logrus.Fields{"account": principal.GetAccount(),
		"request_id": requestId})

	return &ReceptorHttpProxyProbe{logger: logger, metrics: newMetrics()}
}

func (rhpp *ReceptorHttpProxyProbe) sendingMessage(accountNumber, recipient string) {
	rhpp.logger.Infof("Sending message to receptor-gateway - %s:%s\n", accountNumber, recipient)
}

func (rhpp *ReceptorHttpProxyProbe) messageSent(messageID uuid.UUID) {
	rhpp.metrics.receptorProxyMessageSentCounter.Inc()
	rhpp.logger.WithFields(logrus.Fields{"message_id": messageID}).Info("Message sent to receptor-gateway")
}

func (rhpp *ReceptorHttpProxyProbe) failedToSendMessage(errorMsg string, err error) {
	rhpp.metrics.receptorProxyMessageSendFailureCounter.Inc()
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *ReceptorHttpProxyProbe) sendingPing(accountNumber, recipient string) {
	rhpp.logger.Infof("Sending ping message to receptor-gateway - %s:%s\n", accountNumber, recipient)
}

func (rhpp *ReceptorHttpProxyProbe) pingMessageSent() {
	rhpp.metrics.receptorProxyPingMessageSentCounter.Inc()
	rhpp.logger.Info("Ping message sent to receptor-gateway")
}

func (rhpp *ReceptorHttpProxyProbe) closingConnection(accountNumber, recipient string) {
	rhpp.logger.Info("Sending close message to receptor-gateway")
}

func (rhpp *ReceptorHttpProxyProbe) gettingCapabilities(accountNumber, recipient string) {
	rhpp.logger.Info("Getting node capabilities from receptor-gateway")
}

func (rhpp *ReceptorHttpProxyProbe) failedToProcessMessageResponse(errorMsg string, err error) {
	rhpp.metrics.receptorProxyMessageResponseProcessFailureCounter.Inc()
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *ReceptorHttpProxyProbe) failedToSendPingMessage(errorMsg string, err error) {
	rhpp.metrics.receptorProxyPingMessageSendFailureCounter.Inc()
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *ReceptorHttpProxyProbe) failedToProcessPingMessageResponse(errorMsg string, err error) {
	rhpp.metrics.receptorProxyPingMessageResponseProcessFailureCounter.Inc()
	logError(rhpp.logger, err, errorMsg)
}

func (rhpp *ReceptorHttpProxyProbe) failedToCreateHttpRequest(err error) {
	logError(rhpp.logger, err, "Unable to send message.  Failed to create HTTP Request.")
}

func (rhpp *ReceptorHttpProxyProbe) failedToParseHttpResponse(err error) {
	errMsg := "Unable to parse response from receptor-gateway."
	logError(rhpp.logger, err, errMsg)
}

func (rhpp *ReceptorHttpProxyProbe) failedToParseMessageID(err error) {
	errMsg := "Unable to read message id from receptor-gateway"
	logError(rhpp.logger, err, errMsg)
}

func logError(logger *logrus.Entry, err error, errMsg string) {
	logger.WithFields(logrus.Fields{"error": err}).Error(errMsg)
}

type metrics struct {
	receptorProxyMessageSentCounter                   prometheus.Counter
	receptorProxyMessageSendFailureCounter            prometheus.Counter
	receptorProxyMessageResponseProcessFailureCounter prometheus.Counter

	receptorProxyPingMessageSentCounter                   prometheus.Counter
	receptorProxyPingMessageSendFailureCounter            prometheus.Counter
	receptorProxyPingMessageResponseProcessFailureCounter prometheus.Counter
}

func newMetrics() *metrics {
	metrics := new(metrics)

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
