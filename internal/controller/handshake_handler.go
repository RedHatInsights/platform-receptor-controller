package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type HandshakeHandler struct {
	AccountNumber string

	Transport *Transport

	Receptor                 *ReceptorService
	ResponseReactor          ResponseReactor
	ConnectionMgr            *ConnectionManager
	MessageDispatcherFactory *MessageDispatcherFactory
}

func (hh HandshakeHandler) HandleMessage(ctx context.Context, m protocol.Message) {
	if m.Type() != protocol.HiMessageType {
		hh.Transport.ErrorChannel <- fmt.Errorf("Invalid message type (type: %d): %v", m.Type(), m)
		return
	}

	hiMessage, ok := m.(*protocol.HiMessage)
	if !ok {
		hh.Transport.ErrorChannel <- fmt.Errorf("Unable to convert message into HiMessage")
		return
	}

	log.Printf("**** got hi message!!  %+v", hiMessage)

	responseHiMessage := protocol.HiMessage{Command: "HI", ID: hh.Receptor.NodeID}

	ctx, cancel := context.WithTimeout(ctx, time.Second*10) // FIXME:  add a configurable timeout
	defer cancel()

	select {
	case <-ctx.Done():
		log.Println("Request cancelled during handshake. Error: ", ctx.Err())
		return
	case hh.Transport.ControlChannel <- &responseHiMessage: // FIXME:  Why a pointer here??
		break
	}

	// FIXME:  What if this account number and node id are already registered?
	//  abort the connection??

	hh.Receptor.RegisterConnection(hiMessage.ID, hiMessage.Metadata)

	hh.ConnectionMgr.Register(hh.AccountNumber, hiMessage.ID, hh.Receptor)

	disconnectHandler := DisconnectHandler{
		AccountNumber: hh.AccountNumber,
		NodeID:        hiMessage.ID,
		ConnectionMgr: hh.ConnectionMgr,
	}
	hh.ResponseReactor.RegisterDisconnectHandler(disconnectHandler)

	routeTableHandler := RouteTableHandler{
		Receptor:  hh.Receptor,
		Transport: hh.Transport,
	}
	hh.ResponseReactor.RegisterHandler(protocol.RouteTableMessageType, routeTableHandler)

	payloadHandler := PayloadHandler{AccountNumber: hh.AccountNumber,
		Receptor:  hh.Receptor,
		Transport: hh.Transport,
	}
	hh.ResponseReactor.RegisterHandler(protocol.PayloadMessageType, payloadHandler)

	/**** FIXME:
	         1) I think we need to build a Receptor service object that gets created here.
	            The MessageDispatcher likely needs to be created by the Receptor service object
	         2) The MessageDispatcher needs to be disabled until we split the service apart.

		// Start the message dispatcher
		messageDispatcher := hh.MessageDispatcherFactory.NewDispatcher(hh.AccountNumber, hiMessage.ID)
		messageDispatcher.StartDispatchingMessages(ctx, hh.Send)
	*/

	return
}
