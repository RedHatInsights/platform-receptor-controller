package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/queue"
	"github.com/RedHatInsights/platform-receptor-controller/receptor/protocol"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type rcClient struct {
	account string

	// socket is the web socket for this client.
	socket *websocket.Conn

	// send is a channel on which messages are sent.
	send chan []byte
}

func (c *rcClient) SendWork(b []byte) {
	c.send <- b
}

func (c *rcClient) DisconnectReceptorNetwork() {
	fmt.Println("DisconnectReceptorNetwork()")
	c.socket.Close()
}

func (c *rcClient) Close() {
	close(c.send)
}

func performHandshake(socket *websocket.Conn) error {
	messageType, r, err := socket.NextReader()
	fmt.Println("messageType:", messageType)
	fmt.Println("WebSocket reader got a message...")
	if err != nil {
		fmt.Println("WebSocket reader got a error...leaving")
		return err
	}

	if messageType != websocket.BinaryMessage {
		fmt.Println("WebSocket reader...invalid type...leaving")
		return errors.New("websocket reader: invalid message type")
	}

	message, err := protocol.ReadMessage(r)
	fmt.Println("Websocket reader message:", message)
	fmt.Println("Websocket reader message type:", message.Type())

	if message.Type() != protocol.HiMessageType {
		fmt.Println("Received incorrect message type!")
		return errors.New("websocket reader: invalid receptor message type")
	}

	hiMessage := message.(*protocol.HiMessage)

	fmt.Printf("Received a hi message from receptor node %s\n", hiMessage.ID)

	fmt.Println("WebSocket writer - sending HI")

	w, err := socket.NextWriter(websocket.BinaryMessage)

	responseHiMessage := protocol.HiMessage{Command: "HI", ID: "node-cloud-receptor-controller"}

	err = protocol.WriteMessage(w, &responseHiMessage)
	if err != nil {
		fmt.Println("WebSocket writer - error!  Closing connection!")
		return err
	}
	w.Close()

	// FIXME:  Should this "node" generate a UUID for its name to avoid collisions
	fmt.Println("WebSocket writer - sent HI")

	return nil
}

func (c *rcClient) read() {
	defer c.socket.Close()

	err := performHandshake(c.socket)
	if err != nil {
		fmt.Println("Error during handshake:", err)
		return
	}

	go c.write()

	go c.consume()

	for {
		fmt.Println("WebSocket reader waiting for message...")
		messageType, r, err := c.socket.NextReader()
		fmt.Println("messageType:", messageType)
		if err != nil {
			fmt.Println("WebSocket reader got a error...leaving")
			return
		}

		message, err := protocol.ReadMessage(r)
		fmt.Printf("Websocket reader message: %+v\n", message)
		fmt.Println("Websocket reader message type:", message.Type())
	}

	fmt.Println("WebSocket reader leaving!")
}

func (c *rcClient) write() {
	defer c.socket.Close()

	fmt.Println("WebSocket writer - Waiting for something to send")
	for msg := range c.send {
		fmt.Println("Websocket writer needs to send msg:", msg)

		me := "node-cloud-receptor-controller"
		routingMessage := protocol.RoutingMessage{Sender: me,
			Recipient: "node-b",
			RouteList: []string{"node-b"},
		}

		messageId, err := uuid.NewUUID()
		// FIXME: handle error

		innerMessage := protocol.InnerEnvelope{
			MessageID:   messageId.String(),
			Sender:      me,
			Recipient:   "node-b",
			MessageType: "directive",
			RawPayload:  "ima payload bro!",
			Directive:   "demo:do_uptime",
			Timestamp:   protocol.Time{time.Now().UTC()},
		}

		payloadMessage := protocol.PayloadMessage{RoutingInfo: &routingMessage, Data: innerMessage}

		w, err := c.socket.NextWriter(websocket.BinaryMessage)
		if err != nil {
			fmt.Println("WebSocket writer - error!  Closing connection!")
			return
		}

		err = protocol.WriteMessage(w, &payloadMessage)
		if err != nil {
			fmt.Println("WebSocket writer - error writing the message!  Closing connection!")
			return
		}
		w.Close()
	}
	fmt.Println("WebSocket writer leaving!")
}

func (c *rcClient) consume() {
	r := queue.InitConsumer(queue.Get())

	defer func() {
		err := r.Close()
		if err != nil {
			fmt.Println("Error closing consumer: ", err)
			return
		}
		fmt.Println("Consumer closed")
	}()

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			fmt.Println("Error reading message: ", err)
			break
		}
		fmt.Printf("Received message from %s-%d [%d]: %s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
		if string(m.Key) == c.account {
			c.SendWork(m.Value)
		} else {
			fmt.Println("Received message but did not send. Account number not found")
		}
	}
}

type ReceptorController struct {
	connectionMgr *ConnectionManager
	router        *http.ServeMux
}

func newReceptorController(cm *ConnectionManager, r *http.ServeMux) *ReceptorController {
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

func (rc *ReceptorController) routes() {
	rc.router.HandleFunc("/receptor-controller", rc.handleWebSocket())
}

func (rc *ReceptorController) handleWebSocket() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		// 1) Authenticate the connection
		// 2) Verify they are a paying customer
		// 3) Register account with ConnectionManager
		// 4) Start processing messages

		/*
			username, password, ok := req.BasicAuth()
			fmt.Println("username:", username)
			fmt.Println("password:", password)
			fmt.Println("ok:", ok)
			if ok == false {
				log.Println("Failed basic auth")
				return
			}
		*/
		username := "01"

		socket, err := upgrader.Upgrade(w, req, nil)
		fmt.Println("WebSocket server - got a connection, account #", username)
		if err != nil {
			log.Fatal("ServeHTTP:", err)
			return
		}

		client := &rcClient{
			account: username, // FIXME:  for now the username from basic auth is the account
			socket:  socket,
			send:    make(chan []byte, messageBufferSize),
		}

		rc.connectionMgr.Register(client.account, client)

		// once this go routine exits...notify the chat room of the clients departure...close the send channel
		defer func() {
			rc.connectionMgr.Unregister(client.account)
		}()

		client.read()
	}
}
