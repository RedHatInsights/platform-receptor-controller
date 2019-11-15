package protocol

import (
	//	"bufio"
	"encoding/json"
	//"fmt"
	//	"net"
	"time"
	//"os"
)

/*
   buf.push(json.dumps({
         "cmd": "ROUTE",
         "id": self.receptor.node_id,
         "edges": edges,
         "seen": seens
     }).encode("utf-8"))
*/

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

type Message struct {
	Command          string    `json:"cmd"`
	Id               string    `json:"id"`
	Expire_timestamp time.Time `json:"expire_time"`
	// b'{"cmd": "HI", "id": "node-b", "expire_time": 1571507551.7103958}\x1b[K'
}

func (m *Message) Marshal() []byte {
	marshalled_msg, _ := json.Marshal(m)
	return marshalled_msg
}

func (m *Message) Unmarshal(buff []byte) {
	_ = json.Unmarshal(buff, m)
}

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

	// b'{"cmd": "ROUTE", "id": "node-b",
	//    "edges": [["node-a", "node-b", 1],
	//              ["3f4b831d-3c50-4230-925f-7cfc7f00bf8b", "node-a", 1],
	//              ["node-a", "node-golang", 1]
	//             ],
	//    "seen": ["node-a", "node-b"]}\x1b[K'
}
