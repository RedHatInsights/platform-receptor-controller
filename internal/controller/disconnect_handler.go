package controller

import (
	"context"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type DisconnectHandler struct {
	AccountNumber string
	NodeID        string
	Receptor      *ReceptorService
	Dispatcher    ResponseDispatcher
	ConnectionMgr *ConnectionManager
}

func (dh DisconnectHandler) HandleMessage(ctx context.Context, m protocol.Message) {
	dh.ConnectionMgr.Unregister(dh.AccountNumber, dh.NodeID)
	log.Println("DisconnectHandler - account unregistered from connection manager")
	return
}
