package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	kafka "github.com/segmentio/kafka-go"
)

type IMessageHandler interface {
	HandleMessage(context.Context, protocol.Message) error // FIXME: don't need error??  need context??
}

type IResponseDispatcher interface {
	RegisterHandler(protocol.NetworkMessageType, IMessageHandler)
}

type ResponseDispatcherFactory struct {
	writer *kafka.Writer
}

func NewResponseDispatcherFactory(writer *kafka.Writer) *ResponseDispatcherFactory {
	return &ResponseDispatcherFactory{
		writer: writer,
	}
}

func (fact *ResponseDispatcherFactory) NewDispatcher(recv chan protocol.Message) *ResponseDispatcher {

	log.Println("Creating a new response dispatcher")
	return &ResponseDispatcher{
		writer:   fact.writer,
		recv:     recv,
		handlers: make(map[protocol.NetworkMessageType]IMessageHandler),
	}
}

type ResponseDispatcher struct {
	writer   *kafka.Writer
	recv     chan protocol.Message
	handlers map[protocol.NetworkMessageType]IMessageHandler
}

func (rd *ResponseDispatcher) RegisterHandler(msgType protocol.NetworkMessageType, handler IMessageHandler) {
	rd.handlers[msgType] = handler
}

func (rd *ResponseDispatcher) Run(ctx context.Context) {
	defer func() {
		log.Println("ResponseDispatcher leaving!")
	}()

	for {
		log.Println("WebSocket writer - Waiting for something to send")

		select {
		case <-ctx.Done():
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

type MessageDispatcherFactory struct {
	readerConfig *queue.ConsumerConfig
}

func NewMessageDispatcherFactory(cfg *queue.ConsumerConfig) *MessageDispatcherFactory {
	return &MessageDispatcherFactory{
		readerConfig: cfg,
	}
}

func (fact *MessageDispatcherFactory) NewDispatcher(account, nodeID string) *MessageDispatcher {
	log.Println("Creating a new work dispatcher")
	r := queue.StartConsumer(fact.readerConfig)
	return &MessageDispatcher{
		account: account,
		nodeID:  nodeID,
		reader:  r,
	}
}

type MessageDispatcher struct {
	account string
	nodeID  string
	reader  *kafka.Reader
}

func (md *MessageDispatcher) GetKey() string {
	return fmt.Sprintf("%s:%s", md.account, md.nodeID)
}

func (md *MessageDispatcher) StartDispatchingMessages(ctx context.Context, c chan<- Message) {
	defer func() {
		err := md.reader.Close()
		if err != nil {
			log.Println("Kafka job reader - error closing consumer: ", err)
			return
		}
		log.Println("Kafka job reader leaving...")
	}()

	for {
		log.Printf("Kafka job reader - waiting on a message from kafka...")
		m, err := md.reader.ReadMessage(ctx)
		if err != nil {
			// FIXME:  do we need to call cancel here??
			log.Println("Kafka job reader - error reading message: ", err)
			break
		}

		log.Printf("Kafka job reader - received message from %s-%d [%d]: %s: %s\n",
			m.Topic,
			m.Partition,
			m.Offset,
			string(m.Key),
			string(m.Value))

		if string(m.Key) == md.GetKey() {
			// FIXME:
			var w Message
			if err := json.Unmarshal(m.Value, &w); err != nil {
				log.Println("Unable to unmarshal message from kafka queue")
				continue
			}
			c <- w
		} else {
			log.Println("Kafka job reader - received message but did not send. Account number not found.")
		}
	}
}
