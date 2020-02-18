package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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
		Account     string      `json:"account"`
		Sender      string      `json:"sender"`
		MessageType string      `json:"message_type"`
		MessageID   string      `json:"message_id"`
		Payload     interface{} `json:"payload"`
		Code        int         `json:"code"`
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
		Account:     rd.account,
		Sender:      payloadMessage.RoutingInfo.Sender,
		MessageID:   messageID,
		MessageType: payloadMessage.Data.MessageType,
		Payload:     payloadMessage.Data.RawPayload,
		Code:        payloadMessage.Data.Code,
	}

	log.Printf("Dispatching response:%+v", responseMessage)

	jsonResponseMessage, err := json.Marshal(responseMessage)
	if err != nil {
		log.Println("JSON marshal of ResponseMessage failed, err:", err)
		return nil
	}

	rd.writer.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(messageID),
			Value: jsonResponseMessage,
		})

	return nil
}

type WorkDispatcherFactory struct {
	reader *kafka.Reader
}

func NewWorkDispatcherFactory(reader *kafka.Reader) *WorkDispatcherFactory {
	return &WorkDispatcherFactory{
		reader: reader,
	}
}

func (fact *WorkDispatcherFactory) NewDispatcher(account, nodeID string) *WorkDispatcher {
	log.Println("Creating a new work dispatcher")
	return &WorkDispatcher{
		account: account,
		nodeID:  nodeID,
		reader:  fact.reader,
	}
}

type WorkDispatcher struct {
	account string
	nodeID  string
	reader  *kafka.Reader
}

func (wd *WorkDispatcher) GetKey() string {
	return fmt.Sprintf("%s:%s", wd.account, wd.nodeID)
}

func (wd *WorkDispatcher) Dispatch(ctx context.Context) {
	fmt.Println("This is when the work dispatcher would consume a job from the jobs kafka topic")
}
