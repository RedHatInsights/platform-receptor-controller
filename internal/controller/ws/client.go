package ws

import (
	"context"
	"log"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"github.com/gorilla/websocket"
)

type rcClient struct {

	// socket is the web socket for this client.
	socket *websocket.Conn

	// send is a channel on which messages are sent.
	send chan protocol.Message

	controlChannel chan protocol.Message

	errorChannel chan error

	// recv is a channel on which responses are sent.
	recv chan protocol.Message

	cancel context.CancelFunc

	config *WebSocketConfig
}

func (c *rcClient) read(ctx context.Context) {
	defer func() {
		c.socket.Close()
		log.Println("WebSocket reader leaving!")
	}()

	c.configurePongHandler()

	for {
		log.Println("WebSocket reader waiting for message...")
		messageType, r, err := c.socket.NextReader()
		log.Println("Websocket reader: got message")
		log.Println("messageType:", messageType)

		if err != nil {
			log.Println("WebSocket reader got a error: ", err)
			return
		}

		message, err := protocol.ReadMessage(r)
		if err != nil {
			log.Println("WebSocket reader got a error: ", err)
			return
		}

		// The read has completed...disable the read deadline
		c.socket.SetReadDeadline(time.Time{})

		log.Printf("Websocket reader message: %+v\n", message)
		log.Println("Websocket reader message type:", message.Type())

		c.recv <- message
	}
}

func (c *rcClient) configurePongHandler() {

	if c.config.PongWait > 0 {
		log.Println("Configuring a pong handler with a deadline of ", c.config.PongWait)
		c.socket.SetReadDeadline(time.Now().Add(c.config.PongWait))

		c.socket.SetPongHandler(func(data string) error {
			log.Println("WebSocket reader - got a pong")
			c.socket.SetReadDeadline(time.Now().Add(c.config.PongWait))
			return nil
		})
	} else {
		log.Println("Pong handler has been disabled")
	}
}

func writeMessage(socket *websocket.Conn, writeWait time.Duration, msg protocol.Message) error {

	socket.SetWriteDeadline(time.Now().Add(writeWait))
	w, err := socket.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	err = protocol.WriteMessage(w, msg)
	if err != nil {
		return err
	}

	w.Close()

	return nil
}

func (c *rcClient) write(ctx context.Context) {

	pingTicker := c.configurePingTicker()

	defer func() {
		c.socket.Close()
		pingTicker.Stop()
		log.Println("WebSocket writer leaving!")
	}()

	for {
		log.Println("WebSocket writer - Waiting for something to send")

		select {
		case <-ctx.Done():
			return
		case err := <-c.errorChannel:
			log.Println("Websocket writer - got an error - shutting down:", err)
			return
		case msg := <-c.controlChannel:
			log.Println("Websocket writer needs to send msg:", msg)

			err := writeMessage(c.socket, c.config.WriteWait, msg)
			if err != nil {
				log.Println("WebSocket writer - error!  Closing connection! err: ", err)
				return
			}
		case msg := <-c.send:
			log.Println("Websocket writer needs to send msg:", msg)

			err := writeMessage(c.socket, c.config.WriteWait, msg)
			if err != nil {
				log.Println("WebSocket writer - error!  Closing connection! err: ", err)
				return
			}
		case <-pingTicker.C:
			log.Println("WebSocket writer - sending PingMessage")
			c.socket.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
			if err := c.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("WebSocket writer - error sending ping message!  Closing connection!")
				return
			}
		}
	}
}

func (c *rcClient) configurePingTicker() *time.Ticker {

	if c.config.PingPeriod > 0 {
		log.Println("Configuring a ping to fire every ", c.config.PingPeriod)
		return time.NewTicker(c.config.PingPeriod)
	} else {
		log.Println("Pings are disabled")
		// To disable sending ping messages, we create a ticker that doesn't ever fire
		ticker := time.NewTicker(40 * 60 * time.Minute)
		ticker.Stop()
		return ticker
	}
}
