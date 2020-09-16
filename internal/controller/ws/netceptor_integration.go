package ws

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	//"github.com/project-receptor/receptor/pkg/backends"
	"github.com/project-receptor/receptor/pkg/netceptor"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type recvResult struct {
	data []byte
	err  error
}

type NetceptorRCBackend struct {
	conn      *websocket.Conn
	closeChan chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

func (nrcbs *NetceptorRCBackend) Start(ctx context.Context) (chan netceptor.BackendSession, error) {
	fmt.Println("*** Starting backend!!")
	sessChan := make(chan netceptor.BackendSession, 1)

	wsSession := &WebsocketSession{
		conn:            nrcbs.conn,
		recvChan:        make(chan *recvResult),
		closeChan:       nrcbs.closeChan,
		closeChanCloser: sync.Once{},
		//sessChan:        sessChan, // HACK??
		ctx:    nrcbs.ctx,
		cancel: nrcbs.cancel,
	}

	go recvChannelizer(nrcbs.conn, wsSession.recvChan)

	//go func() {
	fmt.Println("Sending session to netceptor")
	sessChan <- wsSession
	fmt.Println("Sent session to netceptor")
	//}()

	return sessChan, nil
}

func recvChannelizer(conn *websocket.Conn, recvChan chan *recvResult) {
	// FIXME: this is gonna leak a go routine
	for {
		_, data, err := conn.ReadMessage()
		//fmt.Println("Got data from websocket: ", data)
		recvChan <- &recvResult{
			data: data,
			err:  err,
		}
		fmt.Println("Sent data to netceptor")

		if err != nil {
			fmt.Printf("Error from conn.ReadMessage(): %s\n", err)
			fmt.Println("This avoids the leak")
			return
		}
	}
}

type WebsocketSession struct {
	conn            *websocket.Conn
	recvChan        chan *recvResult
	closeChan       chan struct{}
	closeChanCloser sync.Once
	//sessChan        chan netceptor.BackendSession
	ctx    context.Context
	cancel context.CancelFunc
}

// Send sends data over the session
func (ns *WebsocketSession) Send(data []byte) error {
	//fmt.Println("Sending data: ", data)
	err := ns.conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		return err
	}
	return nil
}

// Recv receives data via the session
func (ns *WebsocketSession) Recv(timeout time.Duration) ([]byte, error) {
	// fmt.Println("Recv()")
	select {
	case rr := <-ns.recvChan:
		//fmt.Println("Recv - got data:", rr.data)
		return rr.data, rr.err
	case <-time.After(timeout):
		// fmt.Println("Recv - timeout")
		return nil, netceptor.ErrTimeout
	}
}

// Close closes the session
func (ns *WebsocketSession) Close() error {
	fmt.Println("Close()")
	if ns.closeChan != nil {
		ns.closeChanCloser.Do(func() {
			close(ns.closeChan)
			ns.closeChan = nil

			//close(ns.sessChan) // This seems like a hack...but i gotta do this to get out of the outter loop

			fmt.Println("Calling cancel()")
			ns.cancel()
		})
	}
	return ns.conn.Close()
}

type NetceptorClient struct {
	ncObj *netceptor.Netceptor
}

func (nc NetceptorClient) SendMessage(ctx context.Context, account string, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {

	recipient = "node-a"
	fmt.Println("***** SEND_MESSAGE: node: ", recipient)
	conn, err := nc.ncObj.DialContext(ctx, recipient, "echo", nil)
	if err != nil {
		fmt.Println("Error during dial: ", err)
		return nil, err
	}

	workCmd := fmt.Sprintf("work submit %s free", recipient)
	fmt.Println("workCmd: ", workCmd)

	n, err := conn.Write([]byte(workCmd))
	if err != nil {
		fmt.Println("Error during write: ", err)
		return nil, err
	}
	fmt.Printf("Write %d bytes\n", n)

	buf := make([]byte, 1024)
	done := false
	for !done {
		n, err = conn.Read(buf)
		if err == io.EOF {
			done = true
			fmt.Println("Done reading from receptor connection")
		}
		if err != nil {
			fmt.Println("Error during reading from receptor connection: ", err)
			return nil, err
		}
		fmt.Println("Read data receptor network:", string(buf))
	}

	// FIXME:
	myUUID, _ := uuid.NewRandom()
	return &myUUID, nil
}

func (nc NetceptorClient) Ping(ctx context.Context, account string, recipient string, route []string) (interface{}, error) {

	fmt.Println("***** PING")
	recipient = "node-b"
	pingCmd := fmt.Sprintf("ping %s", recipient)

	fmt.Println("NOT RUNNING PING - ", pingCmd)

	return struct{}{}, nil
}

func (nc NetceptorClient) Close(context.Context) error {
	return nil
}

func (nc NetceptorClient) GetCapabilities(context.Context) (interface{}, error) {
	return struct{}{}, nil
}
