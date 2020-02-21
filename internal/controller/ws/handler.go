package ws

import (
	"context"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
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
	messageDispatcherFactory  *controller.MessageDispatcherFactory
}

func NewReceptorController(wsc *WebSocketConfig, cm *controller.ConnectionManager, r *mux.Router, rd *controller.ResponseDispatcherFactory, md *controller.MessageDispatcherFactory) *ReceptorController {
	return &ReceptorController{
		connectionMgr:             cm,
		router:                    r,
		config:                    wsc,
		responseDispatcherFactory: rd,
		messageDispatcherFactory:  md,
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
			config:         rc.config,
			account:        rhIdentity.Identity.AccountNumber,
			socket:         socket,
			send:           make(chan controller.Message, messageBufferSize),
			controlChannel: make(chan protocol.Message, messageBufferSize),
			recv:           make(chan protocol.Message, messageBufferSize),
		}

		ctx := req.Context()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		client.cancel = cancel

		// FIXME: Use the ReceptorFactory to create an instance of the Receptor object

		responseDispatcher := rc.responseDispatcherFactory.NewDispatcher(client.recv, client.account, client.node_id)

		//receptor := controller.Receptor{}

		// FIXME: Register the concrete event handlers with the responseDispatcher

		handshakeHandler := controller.HandshakeHandler{ControlChannel: client.controlChannel /*receptor*/}
		responseDispatcher.RegisterHandler(protocol.HiMessageType, handshakeHandler)

		routeTableHandler := controller.RouteTableHandler{ /*receptor*/ }
		responseDispatcher.RegisterHandler(protocol.RouteTableMessageType, routeTableHandler)

		payloadHandler := controller.PayloadHandler{ /*receptor*/ }
		responseDispatcher.RegisterHandler(protocol.PayloadMessageType, payloadHandler)

		go responseDispatcher.Run(ctx)

		// messageDispatcher := rc.messageDispatcherFactory.NewDispatcher(client.account, client.node_id)

		socket.SetReadLimit(rc.config.MaxMessageSize)

		defer socket.Close()

		client.node_id = "peerID"

		rc.connectionMgr.Register(client.account, client.node_id, client)

		// once this go routine exits...notify the connection manager of the clients departure
		defer func() {
			rc.connectionMgr.Unregister(client.account, client.node_id)
			log.Println("Websocket server - account unregistered from connection manager")
		}()

		// Should the client have a 'handler' function that manages the connection?
		// ex. setting up ping pong, timeouts, cleanup, and calling the goroutines
		// go messageDispatcher.StartDispatchingMessages(ctx, client.send)
		go client.write(ctx)
		client.read(ctx)
	}
}
