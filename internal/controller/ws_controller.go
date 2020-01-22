package controller

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 5 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 25 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024

	// FIXME:  Should this "node" generate a UUID for its name to avoid collisions
	receptorControllerNodeId = "node-cloud-receptor-controller"
)

func init() {
	componentName := "WebSocket"
	log.Printf("%s writeWait: %s", componentName, writeWait)
	log.Printf("%s pongWait: %s", componentName, pongWait)
	log.Printf("%s pingPeriod: %s", componentName, pingPeriod)
	log.Printf("%s maxMessageSize: %d", componentName, maxMessageSize)
}

type rcClient struct {
	account string

	node_id string

	// socket is the web socket for this client.
	socket *websocket.Conn

	// send is a channel on which messages are sent.
	send chan Work

	cancel context.CancelFunc
}

func (c *rcClient) SendWork(w Work) {
	c.send <- w
}

func (c *rcClient) DisconnectReceptorNetwork() {
	log.Println("DisconnectReceptorNetwork()")
	c.socket.Close()
}

func (c *rcClient) Close() {
	// FIXME:  Think through this a bit more.  On close, we might need to to try
	// send a CloseMessage to the client??
	c.cancel()
}

func performHandshake(socket *websocket.Conn) (string, error) {

	socket.SetReadDeadline(time.Now().Add(pongWait))

	messageType, r, err := socket.NextReader()
	log.Println("WebSocket reader got a message...")
	if err != nil {
		log.Println("WebSocket reader - error: ", err)
		return "", err
	}

	if messageType != websocket.BinaryMessage {
		log.Printf("WebSocket reader: invalid type, expected %d, got %d", websocket.BinaryMessage, messageType)
		return "", errors.New("websocket reader: invalid message type")
	}

	message, err := protocol.ReadMessage(r)
	if err != nil {
		log.Println("Websocket reader - error reading/parsing message: ", err)
		return "", err
	}
	log.Println("Websocket reader message:", message)
	log.Println("Websocket reader message type:", message.Type())

	if message.Type() != protocol.HiMessageType {
		log.Printf("WebSocket reader: invalid type, expected %d, got %d", protocol.HiMessageType, message.Type())
		return "", errors.New("websocket reader: invalid receptor message type")
	}

	hiMessage, ok := message.(*protocol.HiMessage)
	if ok != true {
		log.Println("Websocket reader - error casting message to HiMessage")
		return "", errors.New("websocket reader: invalid receptor message type")
	}

	log.Printf("Received a hi message from receptor node %s\n", hiMessage.ID)

	log.Println("WebSocket writer - sending HI")

	socket.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := socket.NextWriter(websocket.BinaryMessage)
	if err != nil {
		log.Println("WebSocket writer - error getting next writer: ", err)
		return "", err
	}

	defer w.Close()

	responseHiMessage := protocol.HiMessage{Command: "HI", ID: receptorControllerNodeId}

	err = protocol.WriteMessage(w, &responseHiMessage)
	if err != nil {
		log.Println("WebSocket writer - error writing message: ", err)
		return "", err
	}

	log.Println("WebSocket writer - sent HI")

	return hiMessage.ID, nil
}

func (c *rcClient) read(ctx context.Context) {
	defer c.socket.Close()

	go c.write(ctx)

	// go c.consume(ctx)

	c.socket.SetReadDeadline(time.Now().Add(pongWait))

	c.socket.SetPongHandler(func(data string) error {
		log.Println("WebSocket reader - got a pong")
		c.socket.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		log.Println("WebSocket reader waiting for message...")
		messageType, r, err := c.socket.NextReader()
		log.Println("Websocket reader: got message")
		log.Println("messageType:", messageType)

		if err != nil {
			log.Println("WebSocket reader got a error: ", err)
			return
		}
		log.Println("messageType:", messageType)

		message, err := protocol.ReadMessage(r)
		if err != nil {
			log.Println("WebSocket reader got a error: ", err)
			return
		}
		log.Printf("Websocket reader message: %+v\n", message)
		log.Println("Websocket reader message type:", message.Type())
	}

	log.Println("WebSocket reader leaving!")
}

func (c *rcClient) write(ctx context.Context) {

	ticker := time.NewTicker(pingPeriod)

	defer func() {
		c.socket.Close()
		log.Println("WebSocket writer leaving!")
	}()

	for {
		log.Println("WebSocket writer - Waiting for something to send")

		select {
		case <-ctx.Done():
			return
		case msg := <-c.send:
			log.Println("Websocket writer needs to send msg:", msg)

			payloadMessage, messageID, err := protocol.BuildPayloadMessage(
				receptorControllerNodeId,
				msg.Recipient,
				msg.RouteList,
				"directive",
				msg.Directive,
				msg.Payload)
			log.Printf("Sending PayloadMessage - %s\n", *messageID)

			c.socket.SetWriteDeadline(time.Now().Add(writeWait))
			w, err := c.socket.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Println("WebSocket writer - error!  Closing connection!")
				return
			}

			err = protocol.WriteMessage(w, payloadMessage)
			if err != nil {
				log.Println("WebSocket writer - error writing the message!  Closing connection!")
				return
			}
			w.Close()
		case <-ticker.C:
			log.Println("WebSocket writer - sending PingMessage")
			c.socket.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("WebSocket writer - error sending ping message!  Closing connection!")
				return
			}
		}
	}

	log.Println("WebSocket writer leaving!")
}

func (c *rcClient) consume(ctx context.Context) {
	r := queue.StartConsumer(queue.Get())

	defer func() {
		err := r.Close()
		if err != nil {
			log.Println("Kafka job reader - error closing consumer: ", err)
			return
		}
		log.Println("Kafka job reader leaving...")
	}()

	for {
		log.Printf("Kafka job reader - waiting on a message from kafka...")
		m, err := r.ReadMessage(ctx)
		if err != nil {
			// FIXME:  do we need to call cancel here??
			log.Println("Kafka job reader - error reading message: ", err)
			break
		}

		log.Printf("Kafka job reader - received message from %s-%d [%d]: %s = %s\n",
			m.Topic,
			m.Partition,
			m.Offset,
			string(m.Key),
			string(m.Value))

		if string(m.Key) == c.account {
			// FIXME:
			w := Work{}
			c.SendWork(w)
		} else {
			log.Println("Kafka job reader - received message but did not send. Account number not found.")
		}
	}
}

type ReceptorController struct {
	connectionMgr *ConnectionManager
	router        *mux.Router
}

func NewReceptorController(cm *ConnectionManager, r *mux.Router) *ReceptorController {
	return &ReceptorController{
		connectionMgr: cm,
		router:        r,
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

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
			send:    make(chan Work, messageBufferSize),
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
