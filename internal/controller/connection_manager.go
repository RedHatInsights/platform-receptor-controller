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

type ConnectionManager struct {
	connections map[string]map[string]Receptor
	sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]map[string]Receptor),
	}
}

func (cm *ConnectionManager) Register(account string, node_id string, client Receptor) {
	cm.Lock()
	defer cm.Unlock()
	_, exists := cm.connections[account]
	if exists == true {
		cm.connections[account][node_id] = client
	} else {
		cm.connections[account] = make(map[string]Receptor)
		cm.connections[account][node_id] = client
	}
	log.Printf("Registered a connection (%s, %s)", account, node_id)
}

func (cm *ConnectionManager) Unregister(account string, node_id string) {
	cm.Lock()
	defer cm.Unlock()
	_, exists := cm.connections[account]
	if exists == false {
		return
	} else {
		delete(cm.connections[account], node_id)

		if len(cm.connections[account]) == 0 {
			delete(cm.connections, account)
		}
	}
	log.Printf("Unregistered a connection (%s, %s)", account, node_id)
}

func (cm *ConnectionManager) GetConnection(account string, node_id string) Receptor {
	var conn Receptor

	cm.RLock()
	defer cm.RUnlock()
	_, exists := cm.connections[account]
	if exists == false {
		return nil
	}

	conn, exists = cm.connections[account][node_id]
	if exists == false {
		return nil
	}

	return conn
}

func (cm *ConnectionManager) GetConnectionsByAccount(account string) map[string]Receptor {
	cm.RLock()
	defer cm.RUnlock()

	retVal := make(map[string]Receptor)

	_, exists := cm.connections[account]
	if exists == false {
		return retVal
	}

	for k, v := range cm.connections[account] {
		retVal[k] = v
	}

	return retVal
}

func (cm *ConnectionManager) GetAllConnections() map[string]map[string]Receptor {
	cm.RLock()
	defer cm.RUnlock()

	retVal := make(map[string]map[string]Receptor)

	for accountNumber, accountMap := range cm.connections {
		retVal[accountNumber] = make(map[string]Receptor)
		log.Println("\taccountNumber:", accountNumber)
		for nodeID, receptorObj := range accountMap {
			log.Println("\t\tnodeID:", nodeID)
			retVal[accountNumber][nodeID] = receptorObj
		}
	}

	return retVal
}
