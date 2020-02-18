package ws

import (
	"context"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

type ReceptorController struct {
	connectionMgr             *controller.ConnectionManager
	router                    *mux.Router
	config                    *WebSocketConfig
	responseDispatcherFactory *controller.ResponseDispatcherFactory
}

func NewReceptorController(wsc *WebSocketConfig, cm *controller.ConnectionManager, r *mux.Router, rd *controller.ResponseDispatcherFactory) *ReceptorController {
	return &ReceptorController{
		connectionMgr:             cm,
		router:                    r,
		config:                    wsc,
		responseDispatcherFactory: rd,
	}
}

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}

func (rc *ReceptorController) Routes() {
	router := rc.router.PathPrefix("/wss/receptor-controller").Subrouter()
	router.Use(identity.EnforceIdentity)
	router.HandleFunc("/gateway", rc.handleWebSocket())
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
			send:    make(chan controller.Message, messageBufferSize),
		}

		client.responseDispatcher = rc.responseDispatcherFactory.NewDispatcher(client.account, client.node_id)

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

		client.node_id = peerID

		rc.connectionMgr.Register(client.account, client.node_id, client)

		// once this go routine exits...notify the connection manager of the clients departure
		defer func() {
			rc.connectionMgr.Unregister(client.account, client.node_id)
			log.Println("Websocket server - account unregistered from connection manager")
		}()

		// Should the client have a 'handler' function that manages the connection?
		// ex. setting up ping pong, timeouts, cleanup, and calling the goroutines
		client.read(ctx)
	}
}
