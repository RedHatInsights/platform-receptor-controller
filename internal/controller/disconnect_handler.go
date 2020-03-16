package controller

import (
	"context"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type DisconnectHandler struct {
	AccountNumber string
	NodeID        string
	ConnectionMgr *ConnectionManager
}

func (dh DisconnectHandler) HandleMessage(ctx context.Context, m protocol.Message) {
	dh.ConnectionMgr.Unregister(dh.AccountNumber, dh.NodeID)
	log.Printf("DisconnectHandler - account (%s) / node id (%s) unregistered from connection manager",
		dh.AccountNumber,
		dh.NodeID)
	return
}
