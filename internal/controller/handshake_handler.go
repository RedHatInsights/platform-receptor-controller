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
	Receptor       *Receptor
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

	responseHiMessage := protocol.HiMessage{Command: "HI", ID: hh.Receptor.NodeID}

	// FIXME:  this can block...add a timeout and a select here???
	hh.ControlChannel <- &responseHiMessage // FIXME:  Why a pointer here??

	hh.Receptor.RegisterConnection(hiMessage.ID, hiMessage.Metadata)

	// FIXME:  this shouldn't be required
	hh.Receptor.HandshakeComplete = true

	return nil
}
