package controller

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/sirupsen/logrus"
)

type GatewayConnectionManager struct {
	rm                     RedisInterface
	localConnectionManager ConnectionManager
}

func NewGatewayConnectionManager(rm RedisInterface, cm ConnectionManager) ConnectionManager {
	return &GatewayConnectionManager{
		rm:                     rm,
		localConnectionManager: cm,
	}
}

func (rcm *GatewayConnectionManager) Register(account string, node_id string, client Receptor) error {
	if rcm.rm.Exists(account, node_id) { // checking connection globally
		logger := logger.Log.WithFields(logrus.Fields{"account": account, "node_id": node_id})
		logger.Warn("Attempting to register duplicate connection")
		metrics.duplicateConnectionCounter.Inc()
		return DuplicateConnectionError{}
	}

	err := rcm.rm.Register(account, node_id)
	if err != nil {
		return err
	}

	err = rcm.localConnectionManager.Register(account, node_id, client)
	if err != nil {
		rcm.Unregister(account, node_id)
		return err
	}

	logger.Log.Printf("Registered a connection (%s, %s)", account, node_id)
	return nil
}

func (rcm *GatewayConnectionManager) Unregister(account string, node_id string) {
	rcm.rm.Unregister(account, node_id)
	rcm.localConnectionManager.Unregister(account, node_id)
	logger.Log.Printf("Unregistered a connection (%s, %s)", account, node_id)
}

func (rcm *GatewayConnectionManager) GetConnection(account string, node_id string) Receptor {
	return rcm.localConnectionManager.GetConnection(account, node_id)
}

func (rcm *GatewayConnectionManager) GetConnectionsByAccount(account string) map[string]Receptor {
	return rcm.localConnectionManager.GetConnectionsByAccount(account)
}

func (rcm *GatewayConnectionManager) GetAllConnections() map[string]map[string]Receptor {
	return rcm.localConnectionManager.GetAllConnections()
}
