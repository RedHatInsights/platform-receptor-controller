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

type frameType int8

const (
	HeaderFrame  frameType = 1
	PayloadFrame frameType = 2
	CommandFrame frameType = 3
)

type messageID [16]byte

func (mid messageID) String() string {
	return fmt.Sprintf("%X-%X-%X-%X-%X",
		mid[0:4],
		mid[4:6],
		mid[6:8],
		mid[8:10],
		mid[10:],
	)
}

type frame struct {
	Type    frameType
	Version byte
	ID      uint32
	Length  uint32
	MsgID   messageID
}

const frameLength int = 26

func (f *frame) unmarshal(buf []byte) error {

	if len(buf) < frameLength {
		return io.ErrUnexpectedEOF
	}

	fb := buf[0:frameLength]

	r := bytes.NewReader(fb)

	fmt.Printf("type(r): %T", r)
	fmt.Println("len(buf): ", len(buf))
	fmt.Println("len(fb): ", len(fb))

	err := binary.Read(r, binary.BigEndian, f)
	if err != nil {
		fmt.Println("failed to read frame:", err)
		return err
	}

	if f.isValidType() != true {
		return fmt.Errorf("invalid frame type")
	}

	fmt.Println("len(buf): ", len(buf))

	dataBuf := make([]byte, f.Length)

	copy(dataBuf, buf[frameLength:])
	fmt.Println(dataBuf)

	return nil
}

func (f *frame) isValidType() bool {
	return f.Type == HeaderFrame || f.Type == PayloadFrame || f.Type == CommandFrame
}

func (f *frame) marshal() ([]byte, error) {
	w := new(bytes.Buffer)

	err := binary.Write(w, binary.BigEndian, f)
	if err != nil {
		fmt.Println("failed to write frame:", err)
		return nil, err
	}

	return w.Bytes(), nil
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
	Command          string      `json:"cmd"`
	Id               string      `json:"id"`
	Expire_timestamp time.Time   `json:"expire_time"`
	Metadata         interface{} `json:"meta"`
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
