package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	kafka "github.com/segmentio/kafka-go"
)

type PayloadHandler struct {
	Account    string
	NodeID     string
	Writer     *kafka.Writer
	ReceptorSM *ReceptorStateMachine
}

func (rd *PayloadHandler) GetKey() string {
	return fmt.Sprintf("%s:%s", rd.Account, rd.NodeID)
}

func (rd PayloadHandler) HandleMessage(ctx context.Context, m protocol.Message /*, receptorID string*/) error {
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
		Account:      rd.Account,
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

	rd.Writer.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(payloadMessage.Data.InResponseTo),
			Value: jsonResponseMessage,
		})

	return nil
}
