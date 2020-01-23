package ws

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"go.uber.org/goleak"

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

func leaks() error {
	return goleak.Find(goleak.IgnoreTopFunction("github.com/onsi/ginkgo/internal/specrunner.(*SpecRunner).registerForInterrupts"))
}

var _ = Describe("WsController", func() {
	var (
		identity string
		wsMux    *mux.Router
		cm       *controller.ConnectionManager
		rc       *ReceptorController
		d        *websocket.Dialer
		header   http.Header
	)

	BeforeEach(func() {
		wsMux = mux.NewRouter()
		cm = controller.NewConnectionManager()
		rc = NewReceptorController(cm, wsMux)
		rc.Routes()

		d = wstest.NewDialer(rc.router)
		identity = `{ "identity": {"account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`
		header = map[string][]string{
			"x-rh-identity": {base64.StdEncoding.EncodeToString([]byte(identity))},
		}
	})

	AfterEach(func() {
		fmt.Println("Checking for leaky goroutines...")
		Eventually(leaks).ShouldNot(HaveOccurred())
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

	Describe("Connecting to the receptor controller with a handshake that takes too long", func() {
		Context("With an open connection and trying to read from the connection", func() {
			It("Should in return receive connection closed error", func() {

				Skip("Skipping for now.  This test needs to be able to configure the websocket timeouts.")

				c, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()

				c.SetReadDeadline(time.Now().Add(2 * time.Second))
				_, _, err = c.NextReader()
				Expect(err).Should(MatchError(&websocket.CloseError{Code: 1006, Text: "unexpected EOF"}))
			})
		})
	})

	Describe("Connecting to the receptor controller and sending work", func() {
		Context("With an open connection", func() {
			It("Should send a ping directive", func() {

				Skip("Skipping for now.  This test is broken due to a race condition.")

				c, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()

				nodeID := "TestClient"

				hiMessage := protocol.HiMessage{Command: "HI", ID: nodeID}
				writeSocket(c, &hiMessage)

				_ = readSocket(c, 1)

				// FIXME: The test fails here.  The issue is that the go routine that handles the
				// web socket performs the handshake...which sends a HiMessage back to the client
				// BEFORE the connection is registered with the connection manager.  This means that
				// if the timing is right and this client code runs as a different go routine, then
				// client is nil below.
				client := rc.connectionMgr.GetConnection("540155", nodeID)

				workRequest := controller.Work{MessageID: "123",
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
