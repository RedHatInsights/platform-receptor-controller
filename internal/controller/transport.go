package controller

import (
	"context"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type Transport struct {

	// send is a channel on which messages are sent.
	Send chan Message

	// recv is a channel on which responses are sent.
	Recv chan protocol.Message

	// controlChannel is a channel on which command/control
	// messages are passed to the write side of the websocket...
	// this allows command/control messages to bypass any queued
	// job messages
	ControlChannel chan protocol.Message

	// errorChannel is a channel on which errors are sent
	// to the go routine managing write side of the websocket
	ErrorChannel chan error

	Ctx    context.Context
	Cancel context.CancelFunc
}
