package controller

import (
	"context"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	"github.com/sirupsen/logrus"
)

type DisconnectHandler struct {
	AccountNumber string
	NodeID        string
	ConnectionMgr ConnectionRegistrar
	Logger        *logrus.Entry
}

func (dh DisconnectHandler) HandleMessage(ctx context.Context, m protocol.Message) {
	dh.ConnectionMgr.Unregister(dh.AccountNumber, dh.NodeID)
	dh.Logger.Debugf("DisconnectHandler - account (%s) / node id (%s) unregistered from connection manager",
		dh.AccountNumber,
		dh.NodeID)
	return
}
