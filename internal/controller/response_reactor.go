package controller

import (
	"context"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	"github.com/sirupsen/logrus"
)

type MessageHandler interface {
	HandleMessage(context.Context, protocol.Message)
}

type ResponseReactor interface {
	RegisterHandler(protocol.NetworkMessageType, MessageHandler)
	RegisterDisconnectHandler(handler MessageHandler)
	Run(context.Context)
}

type ResponseReactorFactory struct {
}

func NewResponseReactorFactory() *ResponseReactorFactory {
	return &ResponseReactorFactory{}
}

func (fact *ResponseReactorFactory) NewResponseReactor(logger *logrus.Entry, recv <-chan protocol.Message) ResponseReactor {

	logger.Debug("Creating a new response dispatcher")
	return &ResponseReactorImpl{
		recv:     recv,
		handlers: make(map[protocol.NetworkMessageType]MessageHandler),
		logger:   logger,
	}
}

type ResponseReactorImpl struct {
	recv              <-chan protocol.Message
	handlers          map[protocol.NetworkMessageType]MessageHandler
	disconnectHandler MessageHandler
	logger            *logrus.Entry
}

func (rd *ResponseReactorImpl) RegisterHandler(msgType protocol.NetworkMessageType, handler MessageHandler) {
	rd.handlers[msgType] = handler
}

func (rd *ResponseReactorImpl) RegisterDisconnectHandler(handler MessageHandler) {
	rd.disconnectHandler = handler
}

func (rd *ResponseReactorImpl) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if rd.disconnectHandler != nil {
				rd.disconnectHandler.HandleMessage(ctx, nil)
			}
			return
		case msg := <-rd.recv:
			rd.logger.Debugf("Dispatching message (type: %d)", msg.Type())
			rd.logger.Tracef("Dispatching message: %+v", msg)

			handler, exists := rd.handlers[msg.Type()]
			if exists == false {
				rd.logger.Debugf("Unable to dispatch message type (%d) - no suitable handler found", msg.Type())
				continue
			}

			handler.HandleMessage(ctx, msg)
		}
	}

}
