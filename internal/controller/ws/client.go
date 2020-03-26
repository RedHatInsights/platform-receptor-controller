package ws

import (
	"context"
	"time"

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

	log *logrus.Entry
}

func (c *rcClient) read(ctx context.Context) {
	defer func() {
		c.socket.Close()
	}()

	c.configurePongHandler()

	for {
		_, r, err := c.socket.NextReader()

		if err != nil {
			c.log.WithFields(logrus.Fields{"error": err}).Debug("WebSocket reader while getting a reader")
			return
		}

		message, err := protocol.ReadMessage(r)
		if err != nil {
			c.log.WithFields(logrus.Fields{"error": err}).Debug("WebSocket reader while reading receptor message")
			return
		}

		// The read has completed...disable the read deadline
		c.socket.SetReadDeadline(time.Time{})

		c.recv <- message
	}
}

func (c *rcClient) configurePongHandler() {

	if c.config.PongWait > 0 {
		c.log.Debug("Configuring a pong handler with a deadline of ", c.config.PongWait)
		c.socket.SetReadDeadline(time.Now().Add(c.config.PongWait))

		c.socket.SetPongHandler(func(data string) error {
			//logger.log.Debug("WebSocket reader - got a pong")
			c.socket.SetReadDeadline(time.Now().Add(c.config.PongWait))
			return nil
		})
	} else {
		c.log.Debug("Pong handler has been disabled")
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
	}()

	for {

		select {
		case <-ctx.Done():
			return

		case err := <-c.errorChannel:
			c.log.WithFields(logrus.Fields{"error": err}).Debug("Got an error from the sync layer...shutting down")
			return

		case msg := <-c.controlChannel:
			err := writeMessage(c.socket, c.config.WriteWait, msg)
			if err != nil {
				c.log.WithFields(logrus.Fields{"error": err}).Debug("An error occurred while sending a control message")
				return
			}

		case msg := <-c.send:
			err := writeMessage(c.socket, c.config.WriteWait, msg)
			if err != nil {
				c.log.WithFields(logrus.Fields{"error": err}).Debug("An error occurred while sending a message")
				return
			}

		case <-pingTicker.C:
			c.socket.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
			if err := c.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.log.WithFields(logrus.Fields{"error": err}).Debug("An error occurred while sending a ping")
				return
			}
		}
	}
}

func (c *rcClient) configurePingTicker() *time.Ticker {

	if c.config.PingPeriod > 0 {
		c.log.Debug("Configuring a ping to fire every ", c.config.PingPeriod)
		return time.NewTicker(c.config.PingPeriod)
	} else {
		c.log.Debug("Pings are disabled")
		// To disable sending ping messages, we create a ticker that doesn't ever fire
		ticker := time.NewTicker(40 * 60 * time.Minute)
		ticker.Stop()
		return ticker
	}
}
