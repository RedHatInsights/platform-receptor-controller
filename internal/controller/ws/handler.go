package ws

import (
	"context"
	"fmt"
	//"io"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	netcLogger "github.com/project-receptor/receptor/pkg/logger"
	"github.com/project-receptor/receptor/pkg/netceptor"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type ReceptorController struct {
	connectionMgr            controller.ConnectionRegistrar
	router                   *mux.Router
	config                   *config.Config
	responseReactorFactory   *controller.ResponseReactorFactory
	messageDispatcherFactory *controller.MessageDispatcherFactory
	receptorServiceFactory   *controller.ReceptorServiceFactory
}

func NewReceptorController(cfg *config.Config, cm controller.ConnectionRegistrar, r *mux.Router, rd *controller.ResponseReactorFactory, md *controller.MessageDispatcherFactory, rs *controller.ReceptorServiceFactory) *ReceptorController {
	return &ReceptorController{
		connectionMgr:            cm,
		router:                   r,
		config:                   cfg,
		responseReactorFactory:   rd,
		messageDispatcherFactory: md,
		receptorServiceFactory:   rs,
	}
}

func (rc *ReceptorController) Routes() {
	router := rc.router.PathPrefix("/wss/receptor-controller").Subrouter()
	router.Use(logger.AccessLoggerMiddleware, identity.EnforceIdentity)
	router.HandleFunc("/gateway", rc.handleWebSocket()).Methods(http.MethodGet)
}

func (rc *ReceptorController) handleWebSocket() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		netcLogger.SetLogLevel(netcLogger.DebugLevel)
		//netcLogger.SetShowTrace(true)

		upgrader := &websocket.Upgrader{ReadBufferSize: rc.config.SocketBufferSize, WriteBufferSize: rc.config.SocketBufferSize}

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

		ctx := req.Context()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		closeChan := make(chan struct{})

		backend := &NetceptorRCBackend{
			conn:      socket,
			closeChan: closeChan,
			ctx:       ctx,
			cancel:    cancel,
		}

		controllerNode := netceptor.New(ctx, rc.config.ReceptorControllerNodeId, nil)
		fmt.Println("calling AddBackend")
		err = controllerNode.AddBackend(backend, 1.0, nil)
		fmt.Println("called AddBackend")
		if err != nil {
			fmt.Printf("Error adding backend to netceptor: %s\n", err)
			panic("AH!")
		}

		fmt.Println("NEW - controllerNode.Status(): ", controllerNode.Status())

		receptorObj := NetceptorClient{controllerNode}

		// FIXME:
		peerNodeID := "node-b"

		err = rc.connectionMgr.Register(ctx, rhIdentity.Identity.AccountNumber, peerNodeID, receptorObj)
		if err != nil {
			fmt.Println("Error registering receptor connection with connection manager")
		}

		fmt.Println("Blocking main HTTP go routine here")
		controllerNode.BackendWait()
		fmt.Println("main HTTP go routine unblocked...leaving")

		rc.connectionMgr.Unregister(ctx, rhIdentity.Identity.AccountNumber, peerNodeID)

		logger.Info("Closing websocket connection")
	}
}
