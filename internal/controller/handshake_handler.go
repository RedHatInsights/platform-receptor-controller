package controller

import (
	"context"
	"fmt"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type HandshakeHandler struct {
	AccountNumber            string
	Send                     chan<- Message
	ControlChannel           chan protocol.Message
	ErrorChannel             chan error
	Receptor                 *Receptor
	Dispatcher               IResponseDispatcher
	ConnectionMgr            *ConnectionManager
	MessageDispatcherFactory *MessageDispatcherFactory
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

	hh.ConnectionMgr.Register(hh.AccountNumber, hiMessage.ID, nil) // FIXME:

	disconnectHandler := DisconnectHandler{
		AccountNumber: hh.AccountNumber,
		NodeID:        hiMessage.ID,
		ConnectionMgr: hh.ConnectionMgr,
	}
	hh.Dispatcher.RegisterHandler(protocol.DisconnectMessageType, disconnectHandler)

	routeTableHandler := RouteTableHandler{
		Receptor:       hh.Receptor, // FIXME: Dont care...pass in only what is required
		ControlChannel: hh.ControlChannel,
		ErrorChannel:   hh.ErrorChannel,
	}
	hh.Dispatcher.RegisterHandler(protocol.RouteTableMessageType, routeTableHandler)

	payloadHandler := PayloadHandler{AccountNumber: hh.AccountNumber,
		ControlChannel: hh.ControlChannel,
		ErrorChannel:   hh.ErrorChannel,
		Receptor:       hh.Receptor, // FIXME: Dont care...pass in only what is required
	}
	hh.Dispatcher.RegisterHandler(protocol.PayloadMessageType, payloadHandler)

	/**** FIXME:
	         1) I think we need to build a Receptor service object that gets created here.
	            The MessageDispatcher likely needs to be created by the Receptor service object
	         2) The MessageDispatcher needs to be disabled until we split the service apart.

		// Start the message dispatcher
		messageDispatcher := hh.MessageDispatcherFactory.NewDispatcher(hh.AccountNumber, hiMessage.ID)
		messageDispatcher.StartDispatchingMessages(ctx, hh.Send)
	*/

	return nil
}
