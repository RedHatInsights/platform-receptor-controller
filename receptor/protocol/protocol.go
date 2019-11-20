package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type Frame struct {
	Type    byte
	Version byte
	ID      uint32
	Length  uint32
	MsgID   [16]byte
}

func (f *Frame) unmarshal(buf []byte) error {
	r := bytes.NewReader(buf)

	err := binary.Read(r, binary.BigEndian, f)
	if err != nil {
		fmt.Println("failed to read frame:", err)
		return err
	}

	return nil
}

type NetworkMessageType int

const (
	HiMessageType     NetworkMessageType = 1
	RouteMessageType  NetworkMessageType = 2
	WorkMessageType   NetworkMessageType = 3
	StatusMessageType NetworkMessageType = 4
)

type Message interface {
	Type() NetworkMessageType
	//	marshal() ([]byte, error)
	//	unmarshal() ([]byte, error)
}

func ParseMessage(buff []byte) (Message, error) {
	if len(buff) < 1 /* min length */ {
		return nil, io.ErrUnexpectedEOF
	}

	msg_str := string(buff)

	var m Message
	if strings.Contains(msg_str, "HI") {
		fmt.Println("client said HI")
		m = &HiMessage{}
	} else if strings.Contains(msg_str, "ROUTE") {
		fmt.Println("client said ROUTE")
		m = &RouteMessage{}
	} else if strings.Contains(msg_str, "STATUS") {
		m = &StatusMessage{}
	} else {
		fmt.Println("FIXME: unrecognized")
		return nil, fmt.Errorf("unrecognized receptor-network message: %s", msg_str)
	}

	if err := json.Unmarshal(buff, m); err != nil {
		fmt.Println("FIXME: unmarshal failed, err:", err)
		return nil, err
	}

	return m, nil
}

type HiMessage struct {
	Command          string    `json:"cmd"`
	Id               string    `json:"id"`
	Expire_timestamp time.Time `json:"expire_time"`
	// b'{"cmd": "HI", "id": "node-b", "expire_time": 1571507551.7103958}\x1b[K'
}

func (m *HiMessage) Type() NetworkMessageType {
	return HiMessageType
}

/*
func (m *HiMessage) Marshal() []byte {
	return marshalled_msg
}

func (m *HiMessage) Unmarshal(buff []byte) {
	_ = json.Unmarshal(buff, m)
}
*/

/*
type Edge struct {
    Left string
    Right string
    Cost int
}
*/

type RouteMessage struct {
	Command string          `json:"cmd"`
	Id      string          `json:"id"`
	Edges   [][]interface{} `json:"edges"`
	Seen    []string        `json:"seen"`

	// b'{"cmd": "ROUTE",
	//    "id": "node-b",
	//    "edges": [["node-a", "node-b", 1],
	//              ["3f4b831d-3c50-4230-925f-7cfc7f00bf8b", "node-a", 1],
	//              ["node-a", "node-golang", 1]
	//             ],
	//    "seen": ["node-a", "node-b"]}\x1b[K'
}

func (m *RouteMessage) Type() NetworkMessageType {
	return RouteMessageType
}

type StatusMessage struct {
	Id string `json:"id"`
}

func (m *StatusMessage) Type() NetworkMessageType {
	return StatusMessageType
}

type OuterEnvelope struct {
	frame_id   string
	sender     string
	recipient  string
	route_list []string
	inner      InnerEnvelope
}

type InnerEnvelope struct {
	Message_Id   string `json:"message_id"`
	Sender       string `json:"sender"`
	Recipient    string `json:"recipient "`
	Message_type string `json:"message_type"`
	//Timestamp    time.Time `json:"timestamp"`
	Raw_payload string `json:"raw_payload"`
	Directive   string `json:"directive"`
}
