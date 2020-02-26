package controller

import (
	"context"
	"fmt"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type RouteTableHandler struct {
	ControlChannel chan<- protocol.Message
	ErrorChannel   chan<- error
	Receptor       *ReceptorService
}

func (rth RouteTableHandler) HandleMessage(ctx context.Context, m protocol.Message) {

	log.Printf("inside RouteTableHandler...receptor:%+v", rth.Receptor)

	if m.Type() != protocol.RouteTableMessageType {
		log.Printf("Invalid message type (type: %d): %v", m.Type(), m)
		return
	}

	routingTableMessage, ok := m.(*protocol.RouteTableMessage)
	if !ok {
		log.Println("Unable to convert message into RouteTableMessage")
		return
	}

	log.Printf("**** got routing table message!!  %+v", routingTableMessage)

	rth.Receptor.UpdateRoutingTable(
		fmt.Sprintf("%s", routingTableMessage.Edges),
		fmt.Sprintf("%s", routingTableMessage.Seen))

	return
}
