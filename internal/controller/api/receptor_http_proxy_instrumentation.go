package api

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/sirupsen/logrus"
)

type ReceptorHttpProxyProbe struct {
}

func sendingMessage(accountNumber, recipient string) {
	logger.Log.WithFields(logrus.Fields{"account": accountNumber,
		"recipient": recipient}).Error("Sending message to receptor-gateway")
}

func pingingNode(accountNumber, recipient string) {
	logger.Log.WithFields(logrus.Fields{"account": accountNumber,
		"recipient": recipient}).Error("Sending ping message to receptor-gateway")
}

func closingConnection(accountNumber, recipient string) {
	logger.Log.WithFields(logrus.Fields{"account": accountNumber,
		"recipient": recipient}).Error("Sending close message to receptor-gateway")
}

func gettingCapabilities(accountNumber, recipient string) {
	logger.Log.WithFields(logrus.Fields{"account": accountNumber,
		"recipient": recipient}).Error("Getting node capabilities from receptor-gateway")
}

func /*(rhpp *ReceptorHttpProxyProbe)*/ failedToMarshalJsonPayload(err error) {
	logger.Log.WithFields(logrus.Fields{"error": err}).Error("Unable to send message.  Failed to marshal JSON payload.")
}

func /*(rhpp *ReceptorHttpProxyProbe)*/ failedToCreateHttpRequest(err error) {
	logger.Log.WithFields(logrus.Fields{"error": err}).Error("Unable to send message.  Failed to create HTTP Request.")
}

func /*(rhpp *ReceptorHttpProxyProbe)*/ failedToParseHttpResponse(err error) {
	errMsg := "Unable to parse response from receptor-gateway."
	logger.Log.WithFields(logrus.Fields{"error": err}).Error(errMsg)
}

func /*(rhpp *ReceptorHttpProxyProbe)*/ failedToParseMessageID(err error) {
	errMsg := "Unable to read message id from receptor-gateway"
	logger.Log.WithFields(logrus.Fields{"error": err}).Error(errMsg)
}
