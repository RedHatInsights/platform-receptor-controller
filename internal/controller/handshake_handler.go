package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	"github.com/sirupsen/logrus"
)

type HandshakeHandler struct {
	AccountNumber            string
	NodeID                   string
	Transport                *Transport
	ReceptorServiceFactory   *ReceptorServiceFactory
	ResponseReactor          ResponseReactor
	ConnectionMgr            ConnectionRegistrar
	MessageDispatcherFactory *MessageDispatcherFactory
	Logger                   *logrus.Entry
}

func (hh HandshakeHandler) HandleMessage(ctx context.Context, m protocol.Message) {
	if m.Type() != protocol.HiMessageType {
		hh.Transport.ErrorChannel <- ReceptorErrorMessage{
			AccountNumber: hh.AccountNumber,
			Error:         fmt.Errorf("Invalid message type (type: %d): %v", m.Type(), m),
		}
		return
	}

	hiMessage, ok := m.(*protocol.HiMessage)
	if !ok {
		hh.Transport.ErrorChannel <- ReceptorErrorMessage{
			AccountNumber: hh.AccountNumber,
			Error:         fmt.Errorf("Unable to convert message into HiMessage"),
		}
		return
	}

	hh.Logger = hh.Logger.WithFields(logrus.Fields{"peer_node_id": hiMessage.ID})
	hh.Logger.Info("Received handshake message")

	responseHiMessage := protocol.HiMessage{Command: "HI", ID: hh.NodeID}

	ctx, cancel := context.WithTimeout(ctx, time.Second*10) // FIXME:  add a configurable timeout
	defer cancel()

	select {
	case <-ctx.Done():
		hh.Logger.Info("Request cancelled during handshake. Error: ", ctx.Err())
		return
	case hh.Transport.ControlChannel <- ReceptorMessage{
		AccountNumber: hh.AccountNumber,
		Message:       &responseHiMessage}:
		break
	}

	hh.Logger.Info("Handshake complete")

	receptor := hh.ReceptorServiceFactory.NewReceptorService(
		hh.Logger,
		hh.AccountNumber,
		hh.NodeID)

	receptor.RegisterConnection(hiMessage.ID, hiMessage.Metadata, hh.Transport)

	err := hh.ConnectionMgr.Register(hh.Transport.Ctx, hh.AccountNumber, hiMessage.ID, receptor)
	if err != nil {
		// Abort the connection if this account number and node id are already registered
		hh.Logger.WithFields(logrus.Fields{"error": err}).Infof("Unable to register connection "+
			"(%s:%s) with connection manager.  Closing connection!", hh.AccountNumber, hiMessage.ID)

		hh.Transport.ErrorChannel <- ReceptorErrorMessage{
			AccountNumber: hh.AccountNumber,
			Error:         err}

		return
	}

	disconnectHandler := DisconnectHandler{
		AccountNumber: hh.AccountNumber,
		NodeID:        hiMessage.ID,
		ConnectionMgr: hh.ConnectionMgr,
		Logger:        hh.Logger,
	}
	hh.ResponseReactor.RegisterDisconnectHandler(disconnectHandler)

	routeTableHandler := RouteTableHandler{
		Receptor:  receptor,
		Transport: hh.Transport,
		Logger:    hh.Logger,
	}
	hh.ResponseReactor.RegisterHandler(protocol.RouteTableMessageType, routeTableHandler)

	payloadHandler := PayloadHandler{AccountNumber: hh.AccountNumber,
		Receptor:  receptor,
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
