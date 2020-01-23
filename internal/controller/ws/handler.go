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
	connectionMgr *controller.ConnectionManager
	router        *mux.Router
}

func NewReceptorController(cm *controller.ConnectionManager, r *mux.Router) *ReceptorController {
	return &ReceptorController{
		connectionMgr: cm,
		router:        r,
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
			account: rhIdentity.Identity.AccountNumber,
			socket:  socket,
			send:    make(chan controller.Work, messageBufferSize),
		}

		ctx := req.Context()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		client.cancel = cancel

		socket.SetReadLimit(maxMessageSize)

		defer socket.Close()

		peerID, err := performHandshake(client.socket)
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

		client.read(ctx)
	}
}
