package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/posener/wstest"
)

func readSocket(c *websocket.Conn, mt protocol.NetworkMessageType) protocol.Message {
	mtype, r, _ := c.NextReader()
	Expect(mtype).Should(Equal(websocket.BinaryMessage))

	fmt.Println("TestClient reading response from receptor-controller")
	m, _ := protocol.ReadMessage(r)
	Expect(m.Type()).Should(Equal(mt))

	return m
}

func writeSocket(c *websocket.Conn, message protocol.Message) {
	w, err := c.NextWriter(websocket.BinaryMessage)
	Expect(err).NotTo(HaveOccurred())

	fmt.Println("TestClient writing to receptor-controller")
	err = protocol.WriteMessage(w, message)
	Expect(err).NotTo(HaveOccurred())
	w.Close()
}

var _ = Describe("WsController", func() {
	var (
		identity string
		wsMux    *mux.Router
		cm       *ConnectionManager
		rc       *ReceptorController
		d        *websocket.Dialer
		header   http.Header
	)

	BeforeEach(func() {
		wsMux = mux.NewRouter()
		cm = NewConnectionManager()
		rc = NewReceptorController(cm, wsMux)
		rc.Routes()

		d = wstest.NewDialer(rc.router)
		identity = `{ "identity": {"account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`
		header = map[string][]string{
			"x-rh-identity": {base64.StdEncoding.EncodeToString([]byte(identity))},
		}
	})

	Describe("Connecting to the receptor controller", func() {
		Context("With a valid identity header", func() {
			It("Should upgrade the connection to a websocket", func() {
				c, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()
			})
		})
		Context("With a missing identity header", func() {
			It("Should not be able to open a connection", func() {
				_, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", nil)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("With an empty identity header", func() {
			It("Should not be able to open a connection", func() {
				id := header["x-rh-identity"]
				id[0] = ""
				header["x-rh-identity"] = id
				_, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", header)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("With a malformed identity header", func() {
			It("Should not be able to open a connection", func() {
				id := header["x-rh-identity"]
				id[0] = `{ "account_number": "540155 }`
				header["x-rh-identity"] = id
				_, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", header)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Connecting to the receptor controller with a handshake", func() {
		Context("With an open connection and sending Hi", func() {
			It("Should in return receive a HiMessage defined in protocol pkg", func() {
				c, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()

				hiMessage := protocol.HiMessage{Command: "HI", ID: "TestClient"}
				writeSocket(c, &hiMessage)

				m := readSocket(c, 1)
				Expect(m.Type()).To(Equal(protocol.HiMessageType))
			})
		})
	})

	Describe("Connecting to the receptor controller and sending work", func() {
		Context("With an open connection", func() {
			It("Should send a ping directive", func() {
				c, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()

				hiMessage := protocol.HiMessage{Command: "HI", ID: "TestClient"}
				writeSocket(c, &hiMessage)

				_ = readSocket(c, 1)

				client := rc.connectionMgr.GetConnection("01")

				workRequest := Work{MessageID: "123",
					Recipient: "TestClient",
					RouteList: []string{"test-b", "test-a"},
					Payload:   "hello",
					Directive: "receptor:ping"}

				client.SendWork(workRequest)

				m := readSocket(c, 4)      // read response from SendWork request and verify it is a PayloadMessage
				jm, err := json.Marshal(m) // m's marshal/unmarshal functions are private and can't be used here

				// FIXME: Need a better way to verify the message
				Expect(string(jm)).To(ContainSubstring("\"sender\":\"node-cloud-receptor-controller\""))
				Expect(string(jm)).To(ContainSubstring("\"recipient\":\"TestClient\""))
				Expect(string(jm)).To(ContainSubstring("\"directive\":\"receptor:ping\""))
			})
		})
	})
})
