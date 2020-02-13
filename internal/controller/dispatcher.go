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

type ResponseDispatcherFactory struct {
	writer *kafka.Writer
}

func NewResponseDispatcherFactory(writer *kafka.Writer) *ResponseDispatcherFactory {
	return &ResponseDispatcherFactory{
		writer: writer,
	}
}

func (fact *ResponseDispatcherFactory) NewDispatcher(account, nodeID string) *ResponseDispatcher {
	log.Println("Creating a new response dispatcher")
	return &ResponseDispatcher{
		account: account,
		nodeID:  nodeID,
		writer:  fact.writer,
	}
}

type ResponseDispatcher struct {
	account string
	nodeID  string
	writer  *kafka.Writer
}

func (rd *ResponseDispatcher) GetKey() string {
	return fmt.Sprintf("%s:%s", rd.account, rd.nodeID)
}

func (rd *ResponseDispatcher) Dispatch(ctx context.Context, m protocol.Message, receptorID string) error {
	type ResponseMessage struct {
		Account   string      `json:"account"`
		Sender    string      `json:"sender"`
		MessageID string      `json:"message_id"`
		Payload   interface{} `json:"payload"`
	}

	if m.Type() != protocol.PayloadMessageType {
		log.Printf("Unable to dispatch message (type: %d): %s", m.Type(), m)
		return nil
	}

	payloadMessage, ok := m.(*protocol.PayloadMessage)
	if !ok {
		log.Println("Unable to convert message into PayloadMessage")
		return nil
	}

	// verify this message was meant for this receptor/peer (probably want a uuid here)
	if payloadMessage.RoutingInfo.Recipient != receptorID {
		log.Println("Recieved message that was not intended for this node")
		return nil
	}

	messageID := payloadMessage.Data.InResponseTo

	responseMessage := ResponseMessage{
		Account:   rd.account,
		Sender:    payloadMessage.RoutingInfo.Sender,
		MessageID: messageID,
		Payload:   payloadMessage.Data.RawPayload,
	}

	log.Println("Dispatching response:", responseMessage)

	jsonResponseMessage, err := json.Marshal(responseMessage)
	if err != nil {
		log.Println("JSON marshal of ResponseMessage failed, err:", err)
		return nil
	}

	log.Println("Dispatching response:", jsonResponseMessage)

	rd.writer.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(messageID),
			Value: jsonResponseMessage,
		})

	return nil
}

type WorkDispatcherFactory struct {
	readerConfig *queue.ConsumerConfig
}

func NewWorkDispatcherFactory(cfg *queue.ConsumerConfig) *WorkDispatcherFactory {
	return &WorkDispatcherFactory{
		readerConfig: cfg,
	}
}

func (fact *WorkDispatcherFactory) NewDispatcher(account, nodeID string) *WorkDispatcher {
	log.Println("Creating a new work dispatcher")
	return &WorkDispatcher{
		account:      account,
		nodeID:       nodeID,
		readerConfig: fact.readerConfig,
	}
}

type WorkDispatcher struct {
	account      string
	nodeID       string
	readerConfig *queue.ConsumerConfig
}

func (wd *WorkDispatcher) GetKey() string {
	return fmt.Sprintf("%s:%s", wd.account, wd.nodeID)
}

func (wd *WorkDispatcher) Dispatch(ctx context.Context, c chan<- Work) {
	r := queue.StartConsumer(wd.readerConfig)

	defer func() {
		err := r.Close()
		if err != nil {
			log.Println("Kafka job reader - error closing consumer: ", err)
			return
		}
		log.Println("Kafka job reader leaving...")
	}()

	for {
		log.Printf("Kafka job reader - waiting on a message from kafka...")
		m, err := r.ReadMessage(ctx)
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

		if string(m.Key) == wd.GetKey() {
			// FIXME:
			w := Work{}
			c <- w
		} else {
			log.Println("Kafka job reader - received message but did not send. Account number not found.")
		}
	}
}
