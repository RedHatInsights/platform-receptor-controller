package ws

import (
	"context"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
		logger.Log.Debug("WebSocket reader leaving!")
	}()

	c.configurePongHandler()

	for {
		logger.Log.Debug("WebSocket reader waiting for message...")
		messageType, r, err := c.socket.NextReader()
		logger.Log.Debug("Websocket reader: got message")
		logger.Log.Debug("messageType:", messageType)

		if err != nil {
			logger.Log.WithFields(logrus.Fields{"error": err}).Debug("WebSocket reader while getting a reader")
			return
		}

		message, err := protocol.ReadMessage(r)
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"error": err}).Debug("WebSocket reader while reading receptor message")
			return
		}

		// The read has completed...disable the read deadline
		c.socket.SetReadDeadline(time.Time{})

		logger.Log.Debug("Websocket reader message type:", message.Type())

		c.recv <- message
	}
}

func (c *rcClient) configurePongHandler() {

	if c.config.PongWait > 0 {
		logger.Log.Debug("Configuring a pong handler with a deadline of ", c.config.PongWait)
		c.socket.SetReadDeadline(time.Now().Add(c.config.PongWait))

		c.socket.SetPongHandler(func(data string) error {
			//logger.log.Debug("WebSocket reader - got a pong")
			c.socket.SetReadDeadline(time.Now().Add(c.config.PongWait))
			return nil
		})
	} else {
		logger.Log.Debug("Pong handler has been disabled")
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
		logger.Log.Debug("WebSocket writer leaving!")
	}()

	for {
		logger.Log.Debug("WebSocket writer - Waiting for something to send")

		select {
		case <-ctx.Done():
			return
		case err := <-c.errorChannel:
			logger.Log.WithFields(logrus.Fields{"error": err}).Debug("Got an error from the sync layer...shutting down")
			return
		case msg := <-c.controlChannel:
			logger.Log.Debug("Websocket writer needs to send control message")

			err := writeMessage(c.socket, c.config.WriteWait, msg)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{"error": err}).Debug("WebSocket writer - caught an error")
				return
			}
		case msg := <-c.send:
			logger.Log.Debug("Websocket writer needs to send message")

			err := writeMessage(c.socket, c.config.WriteWait, msg)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{"error": err}).Debug("WebSocket writer - caught an error")
				return
			}
		case <-pingTicker.C:
			logger.Log.Debug("WebSocket writer - sending PingMessage")
			c.socket.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
			if err := c.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Log.Debug("WebSocket writer - error sending ping message!")
				return
			}
		}
	}
}

func (c *rcClient) configurePingTicker() *time.Ticker {

	if c.config.PingPeriod > 0 {
		logger.Log.Debug("Configuring a ping to fire every ", c.config.PingPeriod)
		return time.NewTicker(c.config.PingPeriod)
	} else {
		logger.Log.Debug("Pings are disabled")
		// To disable sending ping messages, we create a ticker that doesn't ever fire
		ticker := time.NewTicker(40 * 60 * time.Minute)
		ticker.Stop()
		return ticker
	}
}
