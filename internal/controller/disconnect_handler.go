package controller

import (
	"context"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type DisconnectHandler struct {
	AccountNumber  string
	NodeID         string
	ControlChannel chan protocol.Message
	ErrorChannel   chan error
	Receptor       *Receptor
	Dispatcher     IResponseDispatcher
	ConnectionMgr  *ConnectionManager
}

func (dh DisconnectHandler) HandleMessage(ctx context.Context, m protocol.Message) error {
	dh.ConnectionMgr.Unregister(dh.AccountNumber, dh.NodeID)
	log.Println("DisconnectHandler - account unregistered from connection manager")
	return nil
}
