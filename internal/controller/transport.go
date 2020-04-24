package controller

import (
	"context"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type ReceptorMessage struct {
	AccountNumber string
	Message       protocol.Message
}

type ReceptorErrorMessage struct {
	AccountNumber string
	Error         error
}

type Transport struct {

	// send is a channel on which messages are sent.
	Send chan ReceptorMessage

	// recv is a channel on which responses are sent.
	Recv chan protocol.Message

	// controlChannel is a channel on which command/control
	// messages are passed to the write side of the websocket...
	// this allows command/control messages to bypass any queued
	// job messages
	ControlChannel chan ReceptorMessage

	// errorChannel is a channel on which errors are sent
	// to the go routine managing write side of the websocket
	ErrorChannel chan ReceptorErrorMessage

	Ctx    context.Context
	Cancel context.CancelFunc
}
