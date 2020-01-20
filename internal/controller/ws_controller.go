package controller

import (
	"errors"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

type rcClient struct {
	account string

	node_id string

	// socket is the web socket for this client.
	socket *websocket.Conn

	// send is a channel on which messages are sent.
	send chan Work
}

func (c *rcClient) SendWork(w Work) {
	c.send <- w
}

func (c *rcClient) DisconnectReceptorNetwork() {
	log.Println("DisconnectReceptorNetwork()")
	c.socket.Close()
}

func (c *rcClient) Close() {
	close(c.send)
}

func performHandshake(socket *websocket.Conn) (string, error) {
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

	w, err := socket.NextWriter(websocket.BinaryMessage)
	if err != nil {
		log.Println("WebSocket writer - error getting next writer: ", err)
		return "", err
	}

	defer w.Close()

	// FIXME:  Should this "node" generate a UUID for its name to avoid collisions
	responseHiMessage := protocol.HiMessage{Command: "HI", ID: "node-cloud-receptor-controller"}

	err = protocol.WriteMessage(w, &responseHiMessage)
	if err != nil {
		log.Println("WebSocket writer - error writing message: ", err)
		return "", err
	}

	log.Println("WebSocket writer - sent HI")

	return hiMessage.ID, nil
}

func (c *rcClient) read() {
	defer c.socket.Close()

	go c.write()

	// go c.consume()

	for {
		log.Println("WebSocket reader waiting for message...")
		messageType, r, err := c.socket.NextReader()
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

func (c *rcClient) write() {
	defer c.socket.Close()

	log.Println("WebSocket writer - Waiting for something to send")
	for msg := range c.send {
		log.Println("Websocket writer needs to send msg:", msg)

		sender := "node-cloud-receptor-controller"

		payloadMessage, messageID, err := protocol.BuildPayloadMessage(sender,
			msg.Recipient,
			msg.RouteList,
			"directive",
			msg.Directive,
			msg.Payload)
		if err != nil {
			log.Println("Websocket writer - error building payload: ", err)
			return
		}

		log.Printf("Sending PayloadMessage - %s\n", *messageID)

		w, err := c.socket.NextWriter(websocket.BinaryMessage)
		if err != nil {
			log.Println("WebSocket writer - error getting next writer: ", err)
			return
		}

		err = protocol.WriteMessage(w, payloadMessage)
		if err != nil {
			w.Close()
			log.Println("WebSocket writer - error writing the message: ", err)
			return
		}
		w.Close()
	}
	log.Println("WebSocket writer leaving!")
}

// func (c *rcClient) consume() {
// 	r := queue.StartConsumer(queue.Get())

// 	defer func() {
// 		err := r.Close()
// 		if err != nil {
// 			log.Println("Error closing consumer: ", err)
// 			return
// 		}
// 		log.Println("Consumer closed")
// 	}()

// 	for {
// 		m, err := r.ReadMessage(context.Background())
// 		if err != nil {
// 			log.Println("Error reading message: ", err)
// 			break
// 		}
// 		log.Printf("Received message from %s-%d [%d]: %s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
// 		if string(m.Key) == c.account {
// 			// FIXME:
// 			w := Work{}
// 			c.SendWork(w)
// 		} else {
// 			log.Println("Received message but did not send. Account number not found")
// 		}
// 	}
// }

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

		client.read()
	}
}
