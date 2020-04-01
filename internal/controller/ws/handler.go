package ws

import (
	"context"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
	router.HandleFunc("/gateway", rc.handleWebSocket()).Methods(http.MethodGet)
}

func (rc *ReceptorController) handleWebSocket() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		requestId := request_id.GetReqID(req.Context())
		rhIdentity := identity.Get(req.Context())

		logger := logger.Log.WithFields(logrus.Fields{
			"account":    rhIdentity.Identity.AccountNumber,
			"request_id": requestId,
		})

		metrics.TotalConnectionCounter.Inc()
		metrics.ActiveConnectionCounter.Inc()
		defer metrics.ActiveConnectionCounter.Dec()

		socket, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			logger.WithFields(logrus.Fields{"error": err}).Error("Upgrading to a websocket connection failed")
			return
		}

		logger.Info("Accepted websocket connection")

		client := &rcClient{
			config:         rc.config,
			socket:         socket,
			send:           make(chan protocol.Message, messageBufferSize),
			controlChannel: make(chan protocol.Message, messageBufferSize),
			errorChannel:   make(chan error),
			recv:           make(chan protocol.Message, messageBufferSize),
			logger:         logger,
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

		responseReactor := rc.responseReactorFactory.NewResponseReactor(logger, transport.Recv)

		handshakeHandler := controller.HandshakeHandler{
			Transport:                transport,
			ReceptorServiceFactory:   rc.receptorServiceFactory,
			ResponseReactor:          responseReactor,
			AccountNumber:            rhIdentity.Identity.AccountNumber,
			NodeID:                   rc.config.ReceptorControllerNodeId,
			ConnectionMgr:            rc.connectionMgr,
			MessageDispatcherFactory: rc.messageDispatcherFactory,
			Logger:                   logger,
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

		logger.Info("Closing websocket connection")
	}
}
