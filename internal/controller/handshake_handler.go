package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	"github.com/sirupsen/logrus"
)

type HandshakeHandler struct {
	AccountNumber string

	Transport *Transport

	Receptor                 *ReceptorService
	ResponseReactor          ResponseReactor
	ConnectionMgr            *ConnectionManager
	MessageDispatcherFactory *MessageDispatcherFactory
	Logger                   *logrus.Entry
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

	responseHiMessage := protocol.HiMessage{Command: "HI", ID: hh.Receptor.NodeID}

	ctx, cancel := context.WithTimeout(ctx, time.Second*10) // FIXME:  add a configurable timeout
	defer cancel()

	select {
	case <-ctx.Done():
		hh.Logger.Info("Request cancelled during handshake. Error: ", ctx.Err())
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
		Logger:        hh.Logger,
	}
	hh.ResponseReactor.RegisterDisconnectHandler(disconnectHandler)

	routeTableHandler := RouteTableHandler{
		Receptor:  hh.Receptor,
		Transport: hh.Transport,
		Logger:    hh.Logger,
	}
	hh.ResponseReactor.RegisterHandler(protocol.RouteTableMessageType, routeTableHandler)

	payloadHandler := PayloadHandler{AccountNumber: hh.AccountNumber,
		Receptor:  hh.Receptor,
		Transport: hh.Transport,
		Logger:    hh.Logger,
	}
	hh.ResponseReactor.RegisterHandler(protocol.PayloadMessageType, payloadHandler)

	/**** FIXME: The MessageDispatcher needs to be disabled until we split the service apart.

	// Start the message dispatcher
	messageDispatcher := hh.MessageDispatcherFactory.NewDispatcher(hh.AccountNumber, hiMessage.ID)
	messageDispatcher.StartDispatchingMessages(ctx, hh.Send)
	*/

	return
}
