package controller

import (
	"sync"
)

type Work struct {
	MessageID string
	Recipient string
	RouteList []string
	Payload   interface{}
	Directive string
}

type Client interface {
	SendWork(Work)
	Close()
	DisconnectReceptorNetwork()
}

type ConnectionKey struct {
	Account, NodeID string
}

type ConnectionManager struct {
	connections map[ConnectionKey]Client
	sync.Mutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[ConnectionKey]Client),
	}
}

func (cm *ConnectionManager) Register(account string, node_id string, client Client) {
	key := ConnectionKey{account, node_id}
	cm.Lock()
	cm.connections[key] = client
	cm.Unlock()
}

func (cm *ConnectionManager) Unregister(account string, node_id string) {
	key := ConnectionKey{account, node_id}
	cm.Lock()
	conn, exists := cm.connections[key]
	if exists == false {
		return
	}
	conn.Close()
	delete(cm.connections, key)
	cm.Unlock()
}

func (cm *ConnectionManager) GetConnection(account string, node_id string) Client {
	var conn Client

	key := ConnectionKey{account, node_id}

	cm.Lock()
	conn, _ = cm.connections[key]
	cm.Unlock()

	return conn
}
