package controller

import (
	"sync"

	"github.com/google/uuid"
)

type Receptor interface {
	SendMessage(string, []string, interface{}, string) (*uuid.UUID, error)
	Close()
	DisconnectReceptorNetwork()
}

type ConnectionKey struct {
	Account, NodeID string
}

type ConnectionManager struct {
	connections map[ConnectionKey]Receptor
	sync.Mutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[ConnectionKey]Receptor),
	}
}

func (cm *ConnectionManager) Register(account string, node_id string, client Receptor) {
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

func (cm *ConnectionManager) GetConnection(account string, node_id string) Receptor {
	var conn Receptor

	key := ConnectionKey{account, node_id}

	cm.Lock()
	conn, _ = cm.connections[key]
	cm.Unlock()

	return conn
}
