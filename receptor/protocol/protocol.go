package protocol

import (
	//"bytes"
	//"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	//"time"
)

var (
	errInvalidMessage = errors.New("invalid message")
)

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
			fmt.Println("unable to read payload frame")
			return nil, err
		}

		if payloadFrame.Type != PayloadFrameType {
			// FIXME: log it
			fmt.Println("invalid frame type...expected payload frame")
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

	messageBuffer, err := message.marshal()
	if err != nil {
		// FIXME: log the error
		fmt.Println("error marshalling message")
		return err
	}

	frameHeader := FrameHeader{Version: 1, ID: 1}

	frameHeader.Type = CommandFrameType

	frameHeader.Length = uint32(len(messageBuffer))

	frameHeaderBuffer, err := frameHeader.marshal()
	if err != nil {
		// FIXME: log the error
		fmt.Println("error marshalling frame header")
		return err
	}
	fmt.Println("len(frameHeaderBuffer):", len(frameHeaderBuffer))

	n, err := w.Write(frameHeaderBuffer)
	if err != nil {
		// FIXME: log the error
		fmt.Println("error writing frame header")
		return err
	}
	fmt.Println("n:", n)

	n, err = w.Write(messageBuffer)
	if err != nil {
		// FIXME: log the error
		fmt.Println("error writing frame header")
		return err
	}
	fmt.Println("n:", n)

	return nil
}

type NetworkMessageType int

const (
	HiMessageType         NetworkMessageType = 1
	RouteTableMessageType NetworkMessageType = 2
	RoutingMessageType    NetworkMessageType = 3
	PayloadMessageType    NetworkMessageType = 4
)

type Message interface {
	Type() NetworkMessageType
	marshal() ([]byte, error)
	unmarshal(b []byte) error
}

func buildCommandMessage(buff []byte) (Message, error) {
	msgString := string(buff)

	var m Message
	if strings.Contains(msgString, "HI") {
		fmt.Println("client said HI")
		m = new(HiMessage)
	} else if strings.Contains(msgString, "ROUTE") {
		fmt.Println("client said ROUTE")
		m = new(RouteTableMessage)
	} else {
		fmt.Printf("FIXME: unrecognized receptor-network message: %s", msgString)
		return nil, fmt.Errorf("unrecognized receptor-network message: %s", msgString)
	}

	return m, nil
}

type HiMessage struct {
	Command         string      `json:"cmd"`
	Id              string      `json:"id"`
	ExpireTimestamp interface{} `json:"expire_time"` // FIXME:
	Metadata        interface{} `json:"meta"`
	// b'{"cmd": "HI", "id": "node-b", "expire_time": 1571507551.7103958}\x1b[K'
}

func (m *HiMessage) Type() NetworkMessageType {
	return HiMessageType
}

func (m *HiMessage) unmarshal(b []byte) error {

	if err := json.Unmarshal(b, m); err != nil {
		fmt.Println("FIXME: unmarshal of HiMessage failed, err:", err)
		return err
	}

	return nil
}

func (m *HiMessage) marshal() ([]byte, error) {

	b, err := json.Marshal(m)

	if err != nil {
		fmt.Println("FIXME: marshal of HiMessage failed, err:", err)
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
		fmt.Println("FIXME: unmarshal failed, err:", err)
		return err
	}

	return nil
}

func (m *RoutingMessage) marshal() ([]byte, error) {

	b, err := json.Marshal(m)

	if err != nil {
		fmt.Println("FIXME: marshal of RoutingMessage failed, err:", err)
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
	Id           string          `json:"id"`
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
		fmt.Println("FIXME: unmarshal failed, err:", err)
		return err
	}

	return nil
}

func (m *RouteTableMessage) marshal() ([]byte, error) {

	b, err := json.Marshal(m)

	if err != nil {
		fmt.Println("FIXME: marshal of RoutingMessage failed, err:", err)
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
		fmt.Println("FIXME: PayloadMessage unmarshal failed, err:", err)
		return err
	}

	return nil
}

func (m *PayloadMessage) marshal() ([]byte, error) {

	b, err := json.Marshal(m.Data)

	if err != nil {
		fmt.Println("FIXME: marshal of PayloadMessage failed, err:", err)
		return nil, err
	}

	return b, nil
}

type InnerEnvelope struct {
	MessageID   string `json:"message_id"`
	Sender      string `json:"sender"`
	Recipient   string `json:"recipient"`
	MessageType string `json:"message_type"`
	//Timestamp    time.Time `json:"timestamp"`
	RawPayload string `json:"raw_payload"`
	Directive  string `json:"directive"`
}
