package controller

import (
	"context"
	"fmt"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type PayloadHandler struct {
	AccountNumber string
	NodeID        string

	Transport *Transport
	Receptor  *ReceptorService
}

func (ph *PayloadHandler) GetKey() string {
	return fmt.Sprintf("%s:%s", ph.AccountNumber, ph.NodeID)
}

func (ph PayloadHandler) HandleMessage(ctx context.Context, m protocol.Message) {

	if m.Type() != protocol.PayloadMessageType {
		log.Printf("Unable to dispatch message (type: %d): %s", m.Type(), m)
		return
	}

	payloadMessage, ok := m.(*protocol.PayloadMessage)
	if !ok {
		log.Println("Unable to convert message into PayloadMessage")
		return
	}

	// verify this message was meant for this receptor/peer (probably want a uuid here)
	if payloadMessage.RoutingInfo.Recipient != ph.Receptor.NodeID {
		log.Println("Recieved message that was not intended for this node")
		return
	}

	ph.Receptor.DispatchResponse(payloadMessage)

	return
}
