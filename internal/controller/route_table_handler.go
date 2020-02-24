package controller

import (
	"context"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type RouteTableHandler struct {
	ReceptorSM *ReceptorStateMachine
}

func (rth RouteTableHandler) HandleMessage(ctx context.Context, m protocol.Message) error {

	log.Printf("inside RouteTableHandler...statemachine:%+v", rth.ReceptorSM)

	if rth.ReceptorSM.handshakeComplete == false {
		log.Println("Received ROUTE message before handshake was complete")
		// FIXME:  send an error on the ErrorChannel and shutdown connection?!?!
		return nil
	}

	if m.Type() != protocol.RouteTableMessageType {
		log.Printf("Invalid message type (type: %d): %v", m.Type(), m)
		return nil
	}

	routingTableMessage, ok := m.(*protocol.RouteTableMessage)
	if !ok {
		log.Println("Unable to convert message into RouteTableMessage")
		return nil
	}

	log.Printf("**** got routing table message!!  %+v", routingTableMessage)

	rth.ReceptorSM.routingTableReceived = true

	return nil
}
