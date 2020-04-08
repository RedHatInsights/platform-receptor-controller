package ws

import (
	"context"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
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

	logger *logrus.Entry

	config *config.ReceptorControllerConfig
}

func (c *rcClient) read(ctx context.Context) {
	defer func() {
		c.socket.Close()
	}()

	c.configurePongHandler()

	for {
		_, r, err := c.socket.NextReader()

		if err != nil {
			c.logger.WithFields(logrus.Fields{"error": err}).Error("Error while getting a reader from the websocket")
			return
		}

		message, err := protocol.ReadMessage(r)
		if err != nil {
			c.logger.WithFields(logrus.Fields{"error": err}).Error("Error while reading receptor message")
			return
		}

		// The read has completed...disable the read deadline
		c.socket.SetReadDeadline(time.Time{})

		metrics.TotalMessagesReceivedCounter.Inc()

		c.logger.Tracef("Received message: %+v", message)

		c.recv <- message
	}
}

func (c *rcClient) configurePongHandler() {

	if c.config.PongWait > 0 {
		c.logger.Debug("Configuring a pong handler with a deadline of ", c.config.PongWait)
		c.socket.SetReadDeadline(time.Now().Add(c.config.PongWait))

		c.socket.SetPongHandler(func(data string) error {
			// c.logger.Debug("Got a pong message")
			c.socket.SetReadDeadline(time.Now().Add(c.config.PongWait))
			return nil
		})
	} else {
		c.logger.Debug("Pong handler has been disabled")
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

	metrics.TotalMessagesSentCounter.Inc()

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
			c.logger.WithFields(logrus.Fields{"error": err}).Error("Received an error from the sync layer")
			c.socket.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, err.Error()))
			// FIXME: is a sleep needed here??
			return

		case msg := <-c.controlChannel:
			c.logger.Tracef("Sending message received from control channel: %+v", msg)
			err := writeMessage(c.socket, c.config.WriteWait, msg)
			if err != nil {
				c.logger.WithFields(logrus.Fields{"error": err}).Error("Error while sending a control message")
				return
			}

		case msg := <-c.send:
			c.logger.Tracef("Sending message received from send channel: %+v", msg)
			err := writeMessage(c.socket, c.config.WriteWait, msg)
			if err != nil {
				c.logger.WithFields(logrus.Fields{"error": err}).Error("Error while sending a message")
				return
			}

		case <-pingTicker.C:
			// c.logger.Debug("Sending a ping message")
			c.socket.SetWriteDeadline(time.Now().Add(c.config.WriteWait))
			if err := c.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.WithFields(logrus.Fields{"error": err}).Error("Error while sending a ping message")
				return
			}
		}
	}
}

func (c *rcClient) configurePingTicker() *time.Ticker {

	if c.config.PingPeriod > 0 {
		c.logger.Debug("Configuring a ping to fire every ", c.config.PingPeriod)
		return time.NewTicker(c.config.PingPeriod)
	} else {
		c.logger.Debug("Pings are disabled")
		// To disable sending ping messages, we create a ticker that doesn't ever fire
		ticker := time.NewTicker(40 * 60 * time.Minute)
		ticker.Stop()
		return ticker
	}
}
