package controller

import (
	"context"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	kafka "github.com/segmentio/kafka-go"
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

func NewResponseReactorFactory(writer *kafka.Writer) *ResponseReactorFactory {
	return &ResponseReactorFactory{}
}

func (fact *ResponseReactorFactory) NewResponseReactor(recv <-chan protocol.Message) ResponseReactor {

	log.Println("Creating a new response dispatcher")
	return &ResponseReactorImpl{
		recv:     recv,
		handlers: make(map[protocol.NetworkMessageType]MessageHandler),
	}
}

type ResponseReactorImpl struct {
	recv              <-chan protocol.Message
	handlers          map[protocol.NetworkMessageType]MessageHandler
	disconnectHandler MessageHandler
}

func (rd *ResponseReactorImpl) RegisterHandler(msgType protocol.NetworkMessageType, handler MessageHandler) {
	rd.handlers[msgType] = handler
}

func (rd *ResponseReactorImpl) RegisterDisconnectHandler(handler MessageHandler) {
	rd.disconnectHandler = handler
}

func (rd *ResponseReactorImpl) Run(ctx context.Context) {
	defer func() {
		log.Println("ResponseReactor leaving!")
	}()

	for {
		log.Println("ResponseReactorImpl - Waiting for something to send")

		select {
		case <-ctx.Done():
			log.Println("**** dispatching disconnect")
			if rd.disconnectHandler != nil {
				rd.disconnectHandler.HandleMessage(ctx, nil)
			}
			return
		case msg := <-rd.recv:
			log.Printf("**** dispatching message (type: %d): %v", msg.Type(), msg)

			handler, exists := rd.handlers[msg.Type()]
			if exists == false {
				log.Printf("Unable to dispatch message type (%d) - no suitable handler found", msg.Type())
				continue
			}

			handler.HandleMessage(ctx, msg)
		}
	}

}
