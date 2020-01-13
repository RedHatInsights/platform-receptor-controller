package controller

import (
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

const validJSON = `{ "identity": {"account_number": "540155", "type": "User", "internal": { "org_id": "1979710" } } }`

func newServer(t *testing.T, useRouting bool) *httptest.Server {
	rc := NewReceptorController(NewConnectionManager(), mux.NewRouter())
	rc.Routes()

	if useRouting == false {
		s := httptest.NewServer(rc.handleWebSocket())
		s.URL = "ws" + strings.TrimPrefix(s.URL, "http")
		return s
	}

	s := httptest.NewServer(rc.router)
	s.URL = "ws" + strings.TrimPrefix(s.URL, "http")
	return s
}

// This test is just to verify that we can upgrade a connection and not have any leaking goroutines after it closes.
func TestHandshake(t *testing.T) {
	defer goleak.VerifyNone(t)

	s := newServer(t, false)
	defer s.Close()

	ws, _, err := websocket.DefaultDialer.Dial(s.URL, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	hiMessage := protocol.HiMessage{Command: "HI", ID: "test-node"}
	w, err := ws.NextWriter(websocket.BinaryMessage)
	if err := protocol.WriteMessage(w, &hiMessage); err != nil {
		t.Fatalf("Test Write failed to write HI: %v", err)
	}
	w.Close()

	messageType, r, err := ws.NextReader()
	if err != nil {
		t.Fatalf("Test Reader failed to read response: %v", err)
	}
	assert.Equal(t, messageType, websocket.BinaryMessage, "The Hi response from the server should be a binary message")

	message, err := protocol.ReadMessage(r)
	if err != nil {
		t.Fatalf("Protocol Reader failed to read the response: %v", err)
	}
	assert.Equal(t, message.Type(), protocol.HiMessageType, "The Hi response should be of type protocol.HiMessageType")
}

func TestRouting(t *testing.T) {
	defer goleak.VerifyNone(t)

	s := newServer(t, true)
	defer s.Close()

	headerString := base64.StdEncoding.EncodeToString([]byte(validJSON))
	header := map[string][]string{
		"x-rh-identity": {headerString},
	}

	// testing good route
	ws1, _, err := websocket.DefaultDialer.Dial(s.URL+"/receptor-controller", header)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws1.Close()

	// testing request with bad url
	ws2, _, err := websocket.DefaultDialer.Dial(s.URL+"/receptor", header)
	if err == nil {
		t.Fatalf("This test should have thrown an error: %v", ws2)
	}

	// testing request with no identity header
	ws3, _, err := websocket.DefaultDialer.Dial(s.URL+"/receptor-controller", nil)
	if err == nil {
		t.Fatalf("This test should have thrown and error: %v", ws3)
	}
}

func TestRead(t *testing.T) {

}
