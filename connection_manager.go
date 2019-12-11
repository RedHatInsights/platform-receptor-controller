package main

import (
	//"fmt"
	"sync"
)

type Work struct {
	MessageID string
	Recipient string
	RouteList []string
	Payload   string
	Directive string
}

type Client interface {
	SendWork(Work)
	Close()
	DisconnectReceptorNetwork()
}

type ConnectionManager struct {
	connections map[string]Client
	sync.Mutex
}

func newConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]Client),
	}
}

func (cm *ConnectionManager) Register(account string, client Client) {
	cm.Lock()
	cm.connections[account] = client
	cm.Unlock()
}

func (cm *ConnectionManager) Unregister(account string) {
	cm.Lock()
	conn, exists := cm.connections[account]
	if exists == false {
		return
	}
	conn.Close()
	delete(cm.connections, account)
	cm.Unlock()
}

func (cm *ConnectionManager) GetConnection(account string) Client {
	var conn Client

	cm.Lock()
	conn, _ = cm.connections[account]
	cm.Unlock()

	return conn
}
