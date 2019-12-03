package main

import (
	"fmt"
	"log"
	"net/http"

	//	"github.com/RedHatInsights/platform-receptor-controller/receptor/protocol"
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

func (c *rcClient) read() {
	defer c.socket.Close()
	for {
		fmt.Println("WebSocket reader waiting for message...")
		_, msg, err := c.socket.ReadMessage()
		fmt.Println("WebSocket reader got a message...")
		if err != nil {
			fmt.Println("WebSocket reader got a error...leaving")
			return
		}

		msgStr := string(msg)

		fmt.Println("WebSocket Client msg:", msgStr)

		/*
			message, err := protocol.ParseMessage(msg)
			if err != nil {
				fmt.Println("Unable to parse receptor message...ignoring...")
				continue
			}

			fmt.Println("message:", message)
		*/
	}

	fmt.Println("WebSocket reader leaving!")
}

func (c *rcClient) write() {
	defer c.socket.Close()

	fmt.Println("WebSocket writer - sending HI")
	// FIXME:  NOT THE RIGHT PLACE...protocol object should be created...it should
	// manage the state/messages that are sent

	// FIXME:  Should this "node" generate a UUID for its name to avoid collisions
	c.socket.WriteMessage(websocket.TextMessage, []byte("HI:node-golang:timestamp"))
	fmt.Println("WebSocket writer - sent HI")

	fmt.Println("WebSocket writer - Waiting for something to send")
	for msg := range c.send {
		err := c.socket.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			fmt.Println("WebSocket writer - WS error...leaving")
			return
		}
	}
	fmt.Println("WebSocket writer leaving!")
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

		username, password, ok := req.BasicAuth()
		fmt.Println("username:", username)
		fmt.Println("password:", password)
		fmt.Println("ok:", ok)
		if ok == false {
			log.Println("Failed basic auth")
			return
		}

		socket, err := upgrader.Upgrade(w, req, nil)
		fmt.Println("WebSocket client - got a connection")
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

		go client.write()

		client.read()
	}
}
