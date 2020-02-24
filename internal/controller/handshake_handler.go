package controller

import (
	"context"
	"fmt"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type HandshakeHandler struct {
	ControlChannel chan protocol.Message
	ErrorChannel   chan error
	ReceptorSM     *ReceptorStateMachine
}

func (hh HandshakeHandler) HandleMessage(ctx context.Context, m protocol.Message) error {
	if m.Type() != protocol.HiMessageType {
		hh.ErrorChannel <- fmt.Errorf("Invalid message type (type: %d): %v", m.Type(), m)
		return nil
	}

	hiMessage, ok := m.(*protocol.HiMessage)
	if !ok {
		hh.ErrorChannel <- fmt.Errorf("Unable to convert message into HiMessage")
		return nil
	}

	log.Printf("**** got hi message!!  %+v", hiMessage)

	// FIXME: pass the read node name over to the client
	responseHiMessage := protocol.HiMessage{Command: "HI", ID: "c.config.ReceptorControllerNodeId"}

	// FIXME:  this can block...add a timeout and a select here???
	hh.ControlChannel <- &responseHiMessage // FIXME:  Why a pointer here??

	hh.ReceptorSM.handshakeComplete = true
	hh.ReceptorSM.peerNodeID = hiMessage.ID

	// FIXME: Move the capabilities parsing into the protocol!!!
	capMaps, ok := hiMessage.Metadata.(map[string]interface{})
	if ok != true {
		log.Println("Unable to parse capabilities")
		// FIXME:  just ignore that there are not any
		return nil
	}

	for name, value := range capMaps {
		if name == "capabilities" {
			log.Printf("*** %s:%s", name, value)
			hh.ReceptorSM.capabilities = value
		}
	}

	return nil
}
