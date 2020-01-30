package ws

import (
	"context"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/segmentio/kafka-go"
)

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

type ReceptorController struct {
	connectionMgr *controller.ConnectionManager
	router        *mux.Router
	config        *WebSocketConfig
	writer        *kafka.Writer
}

func NewReceptorController(wsc *WebSocketConfig, cm *controller.ConnectionManager, r *mux.Router, kw *kafka.Writer) *ReceptorController {
	return &ReceptorController{
		connectionMgr: cm,
		router:        r,
		writer:        kw,
		config:        wsc,
	}
}

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}

func (rc *ReceptorController) Routes() {
	rc.router.HandleFunc("/wss/receptor-controller/gateway", rc.handleWebSocket())
	rc.router.Use(identity.EnforceIdentity)
}

func (rc *ReceptorController) handleWebSocket() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		socket, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		rhIdentity := identity.Get(req.Context())

		log.Println("WebSocket server - got a connection, account #", rhIdentity.Identity.AccountNumber)
		log.Println("All the headers: ", req.Header)

		client := &rcClient{
			config:  rc.config,
			account: rhIdentity.Identity.AccountNumber,
			socket:  socket,
			send:    make(chan controller.Work, messageBufferSize),
			writer:  rc.writer,
		}

		ctx := req.Context()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		client.cancel = cancel

		socket.SetReadLimit(rc.config.MaxMessageSize)

		defer socket.Close()

		peerID, err := client.performHandshake()
		if err != nil {
			log.Println("Error during handshake:", err)
			return
		}

		rc.connectionMgr.Register(client.account, peerID, client)

		// once this go routine exits...notify the connection manager of the clients departure
		defer func() {
			rc.connectionMgr.Unregister(client.account, peerID)
			log.Println("Websocket server - account unregistered from connection manager")
		}()

		// Should the client have a 'handler' function that manages the connection?
		// ex. setting up ping pong, timeouts, cleanup, and calling the goroutines
		client.read(ctx)
	}
}
