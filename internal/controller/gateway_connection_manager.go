package controller

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/go-redis/redis"

	"github.com/sirupsen/logrus"
)

type GatewayConnectionRegistrar struct {
	redisClient              *redis.Client
	localConnectionRegistrar ConnectionRegistrar
}

func NewGatewayConnectionRegistrar(rdc *redis.Client, cm ConnectionRegistrar) ConnectionRegistrar {
	return &GatewayConnectionRegistrar{
		redisClient:              rdc,
		localConnectionRegistrar: cm,
	}
}

func (rcm *GatewayConnectionRegistrar) Register(account string, node_id string, client Receptor) error {
	if ExistsInRedis(rcm.redisClient, account, node_id) { // checking connection globally
		logger := logger.Log.WithFields(logrus.Fields{"account": account, "node_id": node_id})
		logger.Warn("Attempting to register duplicate connection")
		metrics.duplicateConnectionCounter.Inc()
		return DuplicateConnectionError{}
	}

	err := RegisterWithRedis(rcm.redisClient, account, node_id)
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
	UnregisterWithRedis(rcm.redisClient, account, node_id)
	rcm.localConnectionRegistrar.Unregister(account, node_id)
	logger.Log.Printf("Unregistered a connection (%s, %s)", account, node_id)
}
