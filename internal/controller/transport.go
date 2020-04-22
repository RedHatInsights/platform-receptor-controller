package controller

import (
	"context"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
)

type SendMessage struct {
	AccountNumber string
	Message       protocol.Message
}

type SendErrorMessage struct {
	AccountNumber string
	Error         error
}

type Transport struct {

	// send is a channel on which messages are sent.
	Send chan SendMessage

	// recv is a channel on which responses are sent.
	Recv chan protocol.Message

	// controlChannel is a channel on which command/control
	// messages are passed to the write side of the websocket...
	// this allows command/control messages to bypass any queued
	// job messages
	ControlChannel chan SendMessage

	// errorChannel is a channel on which errors are sent
	// to the go routine managing write side of the websocket
	ErrorChannel chan SendErrorMessage

	Ctx    context.Context
	Cancel context.CancelFunc
}
