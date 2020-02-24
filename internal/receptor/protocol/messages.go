package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	errInvalidMessage = errors.New("invalid message")
)

type NetworkMessageType int

const (
	HiMessageType         NetworkMessageType = 1
	RouteTableMessageType NetworkMessageType = 2
	RoutingMessageType    NetworkMessageType = 3
	PayloadMessageType    NetworkMessageType = 4
)

const jsonTimeFormat = "2006-01-02T15:04:05.999999999"

type Message interface {
	Type() NetworkMessageType
	marshal() ([]byte, error)
	unmarshal(b []byte) error
}

func ReadMessage(r io.Reader) (Message, error) {

	f, err := readFrame(r)
	if err != nil {
		return nil, err
	}

	message, err := parseFrameData(r, f.Type, f.Length)
	if f.Type != HeaderFrameType {
		return message, err
	}

	if f.Type == HeaderFrameType {
		// The current frame is a header frame...so the next chunk of
		// data should be a payload frame followed by the payload data

		payloadFrame, err := readFrame(r)
		if err != nil {
			log.Println("unable to read payload frame:", err)
			return nil, err
		}

		if payloadFrame.Type != PayloadFrameType {
			log.Printf("read invalid frame type...expected payload frame '%d' received frame type '%d'",
				PayloadFrameType,
				payloadFrame.Type)
			return nil, errInvalidMessage
		}

		pm, err := parseFrameData(r, payloadFrame.Type, payloadFrame.Length)
		if err != nil {
			return nil, err
		}
		payloadMessage := pm.(*PayloadMessage)
		payloadMessage.RoutingInfo = message.(*RoutingMessage)

		return payloadMessage, err
	}

	return message, err
}

func WriteMessage(w io.Writer, message Message) error {

	if message.Type() == PayloadMessageType {
		return writePayloadMessage(w, message)
	}

	messageBuffer, err := message.marshal()
	if err != nil {
		// FIXME: log the error
		log.Println("error marshalling message")
		return err
	}

	err = writeFrame(w, CommandFrameType, messageBuffer)
	if err != nil {
		return err
	}

	return nil
}

func writePayloadMessage(w io.Writer, message Message) error {
	payloadMessage := message.(*PayloadMessage)
	routingMessageBuffer, err := payloadMessage.RoutingInfo.marshal()
	if err != nil {
		return err
	}

	err = writeFrame(w, HeaderFrameType, routingMessageBuffer)
	if err != nil {
		return err
	}

	payloadDataBuffer, err := payloadMessage.marshal()
	if err != nil {
		return err
	}

	err = writeFrame(w, PayloadFrameType, payloadDataBuffer)
	if err != nil {
		return err
	}

	return nil
}

func buildCommandMessage(buff []byte) (Message, error) {
	msgString := string(buff)

	var m Message

	// FIXME: look into dynamically unmarshalling json
	if strings.Contains(msgString, "HI") {
		m = new(HiMessage)
	} else if strings.Contains(msgString, "ROUTE") {
		m = new(RouteTableMessage)
	} else {
		log.Printf("FIXME: unrecognized receptor-network message: %s", msgString)
		return nil, fmt.Errorf("unrecognized receptor-network message: %s", msgString)
	}

	return m, nil
}

type HiMessage struct {
	Command         string      `json:"cmd"`
	ID              string      `json:"id"`
	ExpireTimestamp interface{} `json:"expire_time"` // FIXME:
	Metadata        interface{} `json:"meta"`
	// b'{"cmd": "HI", "id": "node-b", "expire_time": 1571507551.7103958}\x1b[K'
}

func (m *HiMessage) Type() NetworkMessageType {
	return HiMessageType
}

func (m *HiMessage) unmarshal(b []byte) error {

	if err := json.Unmarshal(b, m); err != nil {
		log.Println("unmarshal of HiMessage failed, err:", err)
		return err
	}

	return nil
}

func (m *HiMessage) marshal() ([]byte, error) {

	b, err := json.Marshal(m)

	if err != nil {
		log.Println("marshal of HiMessage failed, err:", err)
		return nil, err
	}

	return b, nil
}

type RoutingMessage struct {
	Sender    string   `json:"sender"`
	Recipient string   `json:"recipient"`
	RouteList []string `json:"route_list"`
}

func (m *RoutingMessage) Type() NetworkMessageType {
	return RoutingMessageType
}

func (m *RoutingMessage) unmarshal(b []byte) error {

	if err := json.Unmarshal(b, m); err != nil {
		log.Println("unmarshal of RoutingMessage failed, err:", err)
		return err
	}

	return nil
}

func (m *RoutingMessage) marshal() ([]byte, error) {

	b, err := json.Marshal(m)

	if err != nil {
		log.Println("marshal of RoutingMessage failed, err:", err)
		return nil, err
	}

	return b, nil
}

type Edge struct {
	Left  string
	Right string
	Cost  int
}

type RouteTableMessage struct {
	Command      string          `json:"cmd"`
	ID           string          `json:"id"`
	Capabilities interface{}     `json:"capabilities"`
	Groups       interface{}     `json:"groups"`
	Edges        [][]interface{} `json:"edges"`
	Seen         []string        `json:"seen"`

	// b'{"cmd": "ROUTE",
	//    "id": "node-b",
	//    "edges": [["node-a", "node-b", 1],
	//              ["3f4b831d-3c50-4230-925f-7cfc7f00bf8b", "node-a", 1],
	//              ["node-a", "node-golang", 1]
	//             ],
	//    "seen": ["node-a", "node-b"]}\x1b[K'
}

func (m *RouteTableMessage) Type() NetworkMessageType {
	return RouteTableMessageType
}

func (m *RouteTableMessage) unmarshal(b []byte) error {
	if err := json.Unmarshal(b, m); err != nil {
		log.Println("unmarshal of RouteTableMessage failed, err:", err)
		return err
	}

	return nil
}

func (m *RouteTableMessage) marshal() ([]byte, error) {

	b, err := json.Marshal(m)

	if err != nil {
		log.Println("marshal of RouteTableMessage failed, err:", err)
		return nil, err
	}

	return b, nil
}

type PayloadMessage struct {
	RoutingInfo *RoutingMessage
	Data        InnerEnvelope
}

func (m *PayloadMessage) Type() NetworkMessageType {
	return PayloadMessageType
}

func (pm *PayloadMessage) unmarshal(buf []byte) error {
	if err := json.Unmarshal(buf, &pm.Data); err != nil {
		log.Println("unmarshal of PayloadMessage failed, err:", err)
		return err
	}

	return nil
}

func (m *PayloadMessage) marshal() ([]byte, error) {

	b, err := json.Marshal(m.Data)

	if err != nil {
		log.Println("marshal of PayloadMessage failed, err:", err)
		return nil, err
	}

	return b, nil
}

type InnerEnvelope struct {
	MessageID    string      `json:"message_id"`
	Sender       string      `json:"sender"`
	Recipient    string      `json:"recipient"`
	MessageType  string      `json:"message_type"`
	Timestamp    Time        `json:"timestamp"`
	RawPayload   interface{} `json:"raw_payload"`
	Directive    string      `json:"directive"`
	InResponseTo string      `json:"in_response_to"`
	Code         int         `json:"code"`
	Serial       int         `json:"serial"`
}

type Time struct {
	time.Time
}

func (jt Time) MarshalJSON() ([]byte, error) {
	timeString := fmt.Sprintf("\"%s\"", jt.Format(jsonTimeFormat))
	return []byte(timeString), nil
}

func (jt *Time) UnmarshalJSON(b []byte) error {
	timeString := string(b)
	timeString = strings.Trim(timeString, "\"")

	// Python is sending us a timestamp that looks like this "2019-12-06T04:42:10.988383"
	// but Go is expecting something like this "2006-01-02T15:04:05.999999999Z07:00"...
	// we get a failure like: cannot parse "" as "Z07:00"

	parsedTime, err := time.Parse(jsonTimeFormat, timeString)
	if err != nil {
		log.Println("unmarshal of Time failed: ", err)
		return err
	}

	jt.Time = parsedTime

	return nil
}

func BuildPayloadMessage(messageId uuid.UUID, sender string, recipient string, route []string,
	messageType string, directive string, payload interface{}) (Message, *uuid.UUID, error) {
	routingMessage := RoutingMessage{Sender: sender,
		Recipient: recipient,
		RouteList: route,
	}

	innerMessage := InnerEnvelope{
		MessageID:   messageId.String(),
		Sender:      sender,
		Recipient:   recipient,
		MessageType: messageType,
		Directive:   directive,
		RawPayload:  payload,
		Timestamp:   Time{time.Now().UTC()},
	}

	payloadMessage := &PayloadMessage{RoutingInfo: &routingMessage, Data: innerMessage}

	return payloadMessage, &messageId, nil
}
