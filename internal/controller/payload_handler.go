package controller

import (
	"context"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	"github.com/sirupsen/logrus"
)

type PayloadHandler struct {
	AccountNumber string
	NodeID        string

	Transport *Transport
	Receptor  *ReceptorService
	Logger    *logrus.Entry
}

func (ph PayloadHandler) HandleMessage(ctx context.Context, m protocol.Message) {

	if m.Type() != protocol.PayloadMessageType {
		ph.Logger.Infof("Unable to dispatch message (type: %d): %s", m.Type(), m)
		return
	}

	payloadMessage, ok := m.(*protocol.PayloadMessage)
	if !ok {
		ph.Logger.Info("Unable to convert message into PayloadMessage")
		return
	}

	ph.Receptor.DispatchResponse(payloadMessage)

	return
}
