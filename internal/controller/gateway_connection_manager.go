package controller

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/sirupsen/logrus"
)

type GatewayConnectionRegistrar struct {
	rm                       RedisConnector
	localConnectionRegistrar ConnectionRegistrar
}

func NewGatewayConnectionRegistrar(rm RedisConnector, cm ConnectionRegistrar) ConnectionRegistrar {
	return &GatewayConnectionRegistrar{
		rm:                       rm,
		localConnectionRegistrar: cm,
	}
}

func (rcm *GatewayConnectionRegistrar) Register(account string, node_id string, client Receptor) error {
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

	err = rcm.localConnectionRegistrar.Register(account, node_id, client)
	if err != nil {
		rcm.Unregister(account, node_id)
		return err
	}

	logger.Log.Printf("Registered a connection (%s, %s)", account, node_id)
	return nil
}

func (rcm *GatewayConnectionRegistrar) Unregister(account string, node_id string) {
	rcm.rm.Unregister(account, node_id)
	rcm.localConnectionRegistrar.Unregister(account, node_id)
	logger.Log.Printf("Unregistered a connection (%s, %s)", account, node_id)
}
