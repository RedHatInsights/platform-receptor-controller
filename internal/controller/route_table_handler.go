package controller

import (
	"context"
	"fmt"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	"github.com/sirupsen/logrus"
)

type RouteTableHandler struct {
	Receptor  *ReceptorService
	Transport *Transport
	Logger    *logrus.Entry
}

func (rth RouteTableHandler) HandleMessage(ctx context.Context, m protocol.Message) {

	if m.Type() != protocol.RouteTableMessageType {
		rth.Logger.Infof("Invalid message type (type: %d): %v", m.Type(), m)
		return
	}

	routingTableMessage, ok := m.(*protocol.RouteTableMessage)
	if !ok {
		rth.Logger.Info("Unable to convert message into RouteTableMessage")
		return
	}

	rth.Receptor.UpdateRoutingTable(
		fmt.Sprintf("%s", routingTableMessage.Edges),
		fmt.Sprintf("%s", routingTableMessage.Seen))

	return
}
