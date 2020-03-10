package controller

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
)

type Receptor interface {
	SendMessage(context.Context, string, []string, interface{}, string) (*uuid.UUID, error)
	Ping(context.Context, string, []string) (interface{}, error)
	Close()
	DisconnectReceptorNetwork()
	GetCapabilities() interface{}
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
	log.Printf("Registered a connection (%s, %s)", account, node_id)
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
	log.Printf("Unregistered a connection (%s, %s)", account, node_id)
}

func (cm *ConnectionManager) GetConnection(account string, node_id string) Receptor {
	var conn Receptor

	key := ConnectionKey{account, node_id}

	cm.Lock()
	conn, _ = cm.connections[key]
	cm.Unlock()

	return conn
}
