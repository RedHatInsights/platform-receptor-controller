package controller

import (
	"encoding/base64"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	//c "github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	. "github.com/gorilla/websocket"
	"github.com/posener/wstest"
)

var _ = Describe("WsController", func() {
	var (
		identity string
		wsMux    *mux.Router
		cm       *ConnectionManager
		rc       *ReceptorController
		d        *Dialer
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
				c, _, err := d.Dial("ws://localhost:8080/receptor-controller", header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()
			})
		})
		Context("With a missing identity header", func() {
			It("Should not be able to open a connection", func() {
				_, _, err := d.Dial("ws://localhost:8080/receptor-controller", nil)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("With a malformed identity header", func() {
			It("Should not be able to open a connection", func() {
				id := header["x-rh-identity"]
				id[0] = ""
				header["x-rh-identity"] = id
				_, _, err := d.Dial("ws://localhost:8080/receptor-controller", header)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Connecting to the receptor controller with a handshake", func() {
		Context("With an open connection and sending Hi", func() {
			It("Should in return receive a HiMessage defined in protocol pkg", func() {
				hiMessage := protocol.HiMessage{Command: "HI", ID: "test-node"}
				c, _, err := d.Dial("ws://localhost:8080/receptor-controller", header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()

				w, err := c.NextWriter(websocket.BinaryMessage)
				err = protocol.WriteMessage(w, &hiMessage)
				Expect(err).NotTo(HaveOccurred())
				w.Close()

				mtype, r, err := c.NextReader()
				Expect(err).NotTo(HaveOccurred())
				Expect(mtype).Should(Equal(websocket.BinaryMessage))

				m, err := protocol.ReadMessage(r)
				Expect(err).NotTo(HaveOccurred())
				Expect(m.Type()).Should(Equal(protocol.HiMessageType))
			})
		})
	})
})
