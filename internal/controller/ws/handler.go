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
	connectionMgr            *controller.ConnectionManager
	router                   *mux.Router
	config                   *WebSocketConfig
	responseReactorFactory   *controller.ResponseReactorFactory
	messageDispatcherFactory *controller.MessageDispatcherFactory
	receptorServiceFactory   *controller.ReceptorServiceFactory
}

func NewReceptorController(wsc *WebSocketConfig, cm *controller.ConnectionManager, r *mux.Router, rd *controller.ResponseReactorFactory, md *controller.MessageDispatcherFactory, rs *controller.ReceptorServiceFactory) *ReceptorController {
	return &ReceptorController{
		connectionMgr:            cm,
		router:                   r,
		config:                   wsc,
		responseReactorFactory:   rd,
		messageDispatcherFactory: md,
		receptorServiceFactory:   rs,
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
			socket:         socket,
			send:           make(chan protocol.Message, messageBufferSize),
			controlChannel: make(chan protocol.Message, messageBufferSize),
			errorChannel:   make(chan error),
			recv:           make(chan protocol.Message, messageBufferSize),
		}

		ctx := req.Context()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		client.cancel = cancel

		transport := &controller.Transport{
			Send:           client.send,
			Recv:           client.recv,
			ControlChannel: client.controlChannel,
			ErrorChannel:   client.errorChannel,
			Cancel:         client.cancel,
			Ctx:            ctx,
		}

		responseReactor := rc.responseReactorFactory.NewResponseReactor(transport.Recv)

		receptorService := rc.receptorServiceFactory.NewReceptorService(rhIdentity.Identity.AccountNumber,
			rc.config.ReceptorControllerNodeId,
			transport)

		handshakeHandler := controller.HandshakeHandler{
			Transport:                transport,
			Receptor:                 receptorService,
			ResponseReactor:          responseReactor,
			AccountNumber:            rhIdentity.Identity.AccountNumber,
			ConnectionMgr:            rc.connectionMgr,
			MessageDispatcherFactory: rc.messageDispatcherFactory,
		}
		responseReactor.RegisterHandler(protocol.HiMessageType, handshakeHandler)

		go responseReactor.Run(ctx)

		socket.SetReadLimit(rc.config.MaxMessageSize)

		defer socket.Close()

		// Should the client have a 'handler' function that manages the connection?
		// ex. setting up ping pong, timeouts, cleanup, and calling the goroutines
		// go messageDispatcher.StartDispatchingMessages(ctx, client.send)
		go client.write(ctx)
		client.read(ctx)
	}
}
