package ws

import (
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"github.com/segmentio/kafka-go"
	"go.uber.org/goleak"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/posener/wstest"
)

func readSocket(c *websocket.Conn, mt protocol.NetworkMessageType) (protocol.Message, error) {
	mtype, r, err := c.NextReader()
	Expect(mtype).Should(Equal(websocket.BinaryMessage))

	log.Println("TestClient reading response from receptor-controller")
	m, err := protocol.ReadMessage(r)
	Expect(m.Type()).Should(Equal(mt))

	return m, err
}

func writeSocket(c *websocket.Conn, message protocol.Message) error {
	w, err := c.NextWriter(websocket.BinaryMessage)
	Expect(err).NotTo(HaveOccurred())

	log.Println("TestClient writing to receptor-controller")
	err = protocol.WriteMessage(w, message)
	Expect(err).NotTo(HaveOccurred())
	w.Close()
	return err
}

func leaks() error {
	return goleak.Find(goleak.IgnoreTopFunction("github.com/onsi/ginkgo/internal/specrunner.(*SpecRunner).registerForInterrupts"))
}

var _ = Describe("WsController", func() {
	var (
		identity string
		wsMux    *mux.Router
		cr       controller.ConnectionRegistrar
		cfg      *config.Config
		rc       *ReceptorController
		kw       *kafka.Writer
		d        *websocket.Dialer
		header   http.Header
	)

	BeforeEach(func() {
		wsMux = mux.NewRouter()
		cfg = config.GetConfig()
		cr = controller.NewLocalConnectionManager()
		kc := &queue.ConsumerConfig{
			Brokers:        cfg.KafkaBrokers,
			Topic:          cfg.KafkaJobsTopic,
			GroupID:        cfg.KafkaGroupID,
			ConsumerOffset: cfg.KafkaConsumerOffset,
		}
		md := controller.NewMessageDispatcherFactory(kc)
		kw = queue.StartProducer(&queue.ProducerConfig{
			Brokers: cfg.KafkaBrokers,
			Topic:   cfg.KafkaResponsesTopic,
		})
		rd := controller.NewResponseReactorFactory()
		rs := controller.NewReceptorServiceFactory(kw, cfg)
		rc = NewReceptorController(cfg, cr, wsMux, rd, md, rs)
		rc.Routes()

		d = wstest.NewDialer(rc.router)
		identity = `{ "identity": {"account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`
		header = map[string][]string{
			"x-rh-identity": {base64.StdEncoding.EncodeToString([]byte(identity))},
		}
	})

	AfterEach(func() {
		kw.Close()
		log.Println("Checking for leaky goroutines...")
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

				m, _ := readSocket(c, 1)
				Expect(m.Type()).To(Equal(protocol.HiMessageType))
			})
		})
	})

	Describe("Connecting to the receptor controller and sending a routing message", func() {
		Context("With an open connection and successful handshake", func() {
			It("Should send a routing message and close the connection gracefully", func() {
				c, _, err := d.Dial("ws://localhost:8080/wss/receptor-controller/gateway", header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()

				nodeID := "TestClient"

				handshakeMessage := protocol.HiMessage{Command: "HI", ID: nodeID}
				writeSocket(c, &handshakeMessage)

				handshakeResponse, _ := readSocket(c, 1)
				log.Println("Handshake response: ", handshakeResponse)

				seen := []string{"test-node-a", "test-node-b"}
				edges := [][]interface{}{{"node-a", "node-b", 1}, {"3f4b831d-3c50-4230-925f-7cfc7f00bf8b", "node-a", 1}, {"node-a", "node-golang", 1}}

				routingMessage := protocol.RouteTableMessage{
					Command: "ROUTE",
					ID:      nodeID,
					Edges:   edges,
					Seen:    seen,
				}

				writeSocket(c, &routingMessage)

				log.Println("Sent routing message to the gateway...")

				// FIXME: The gateway currently does not respond to a routing message. This will change in the future.
			})
		})
	})

	Describe("Connecting to the receptor controller with duplicate account and node id", func() {
		Context("With an open connection and open a new connection and send Hi with the same account and node id", func() {
			It("Should in return receive an error on the second connection", func() {

				var RC_URL string = "ws://localhost:8080/wss/receptor-controller/gateway"

				c, _, err := d.Dial(RC_URL, header)
				Expect(err).NotTo(HaveOccurred())
				defer c.Close()

				hiMessage := protocol.HiMessage{Command: "HI", ID: "TestClient"}
				writeSocket(c, &hiMessage)

				m, _ := readSocket(c, 1)
				Expect(m.Type()).To(Equal(protocol.HiMessageType))

				// If we got to this point...we have a valid handshake

				// Open a new connection without closing the previously opened connection
				duplicateConnection, _, err := d.Dial(RC_URL, header)
				Expect(duplicateConnection).Should(BeNil())
				Expect(err).Should(HaveOccurred()) // FIXME: Check for an error
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
})
