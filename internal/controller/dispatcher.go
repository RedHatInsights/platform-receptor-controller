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
	HandleMessage(protocol.Message) error // FIXME: don't need error??  need context??
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

func (fact *ResponseDispatcherFactory) NewDispatcher(recv chan protocol.Message, account, nodeID string) *ResponseDispatcher {

	log.Println("Creating a new response dispatcher")
	return &ResponseDispatcher{
		account:  account,
		nodeID:   nodeID,
		writer:   fact.writer,
		recv:     recv,
		handlers: make(map[protocol.NetworkMessageType]IMessageHandler),
	}
}

type ResponseDispatcher struct {
	account  string
	nodeID   string
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

			handler.HandleMessage(msg)
		}
	}

}

type PayloadHandler struct {
	account string
	nodeID  string
	writer  *kafka.Writer
}

func (rd *PayloadHandler) GetKey() string {
	return fmt.Sprintf("%s:%s", rd.account, rd.nodeID)
}

func (rd PayloadHandler) HandleMessage( /*ctx context.Context,*/ m protocol.Message /*, receptorID string*/) error {
	type ResponseMessage struct {
		Account      string      `json:"account"`
		Sender       string      `json:"sender"`
		MessageType  string      `json:"message_type"`
		MessageID    string      `json:"message_id"`
		Payload      interface{} `json:"payload"`
		Code         int         `json:"code"`
		InResponseTo string      `json:"in_response_to"`
		Serial       int         `json:"serial"`
	}

	// FIXME:
	ctx := context.Background()

	if m.Type() != protocol.PayloadMessageType {
		log.Printf("Unable to dispatch message (type: %d): %s", m.Type(), m)
		return nil
	}

	payloadMessage, ok := m.(*protocol.PayloadMessage)
	if !ok {
		log.Println("Unable to convert message into PayloadMessage")
		return nil
	}

	// FIXME:
	/*
		// verify this message was meant for this receptor/peer (probably want a uuid here)
		if payloadMessage.RoutingInfo.Recipient != receptorID {
			log.Println("Recieved message that was not intended for this node")
			return nil
		}
	*/

	responseMessage := ResponseMessage{
		Account:      rd.account,
		Sender:       payloadMessage.RoutingInfo.Sender,
		MessageID:    payloadMessage.Data.MessageID,
		MessageType:  payloadMessage.Data.MessageType,
		Payload:      payloadMessage.Data.RawPayload,
		Code:         payloadMessage.Data.Code,
		InResponseTo: payloadMessage.Data.InResponseTo,
		Serial:       payloadMessage.Data.Serial,
	}

	log.Printf("Dispatching response:%+v", responseMessage)

	jsonResponseMessage, err := json.Marshal(responseMessage)
	if err != nil {
		log.Println("JSON marshal of ResponseMessage failed, err:", err)
		return nil
	}

	rd.writer.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(payloadMessage.Data.InResponseTo),
			Value: jsonResponseMessage,
		})

	return nil
}

type RouteTableHandler struct {
}

func (rd RouteTableHandler) HandleMessage(m protocol.Message) error {
	if m.Type() != protocol.RouteTableMessageType {
		log.Printf("Invalid message type (type: %d): %v", m.Type(), m)
		return nil
	}

	routingTableMessage, ok := m.(*protocol.RouteTableMessage)
	if !ok {
		log.Println("Unable to convert message into RouteTableMessage")
		return nil
	}

	log.Printf("**** got routing table message!!  %+v", routingTableMessage)

	return nil
}

type HiHandler struct {
	ControlChannel chan protocol.Message
}

func (hi HiHandler) HandleMessage(m protocol.Message) error {
	if m.Type() != protocol.HiMessageType {
		log.Printf("Invalid message type (type: %d): %v", m.Type(), m)
		return nil
	}

	hiMessage, ok := m.(*protocol.HiMessage)
	if !ok {
		log.Println("Unable to convert message into HiMessage")
		return nil
	}

	log.Printf("**** got hi message!!  %+v", hiMessage)

	responseHiMessage := protocol.HiMessage{Command: "HI", ID: "c.config.ReceptorControllerNodeId"}

	hi.ControlChannel <- &responseHiMessage // FIXME:  Why a pointer here??

	return nil
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
