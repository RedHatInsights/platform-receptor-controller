package controller

import (
	"context"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/go-redis/redis"

	"github.com/sirupsen/logrus"
)

type GatewayConnectionRegistrar struct {
	redisClient                      *redis.Client
	localConnectionRegistrar         ConnectionRegistrar
	hostname                         string
	activeConnectionRegistrarFactory ActiveConnectionRegistrarFactory
}

func NewGatewayConnectionRegistrar(rdc *redis.Client, cm ConnectionRegistrar, acrf ActiveConnectionRegistrarFactory, host string) ConnectionRegistrar {
	return &GatewayConnectionRegistrar{
		redisClient:                      rdc,
		localConnectionRegistrar:         cm,
		hostname:                         host,
		activeConnectionRegistrarFactory: acrf,
	}
}

// entry does not exist
// entry exists with right hostname
// entry exists with wrong hostname
// - does hostname exist??

func (rcm *GatewayConnectionRegistrar) Register(ctx context.Context, account string, nodeID string, client Receptor) error {
	logger := logger.Log.WithFields(logrus.Fields{"account": account, "nodeID": nodeID})

	if ExistsInRedis(rcm.redisClient, account, nodeID) { // checking connection globally
		logger.Warn("Attempting to register duplicate connection")
		metrics.duplicateConnectionCounter.Inc()
		return DuplicateConnectionError{}
	}

	err := RegisterWithRedis(rcm.redisClient, account, nodeID, rcm.hostname)
	if err != nil {
		return err
	}

	err = rcm.localConnectionRegistrar.Register(ctx, account, nodeID, client)
	if err != nil {
		rcm.Unregister(ctx, account, nodeID)
		return err
	}

	rcm.activeConnectionRegistrarFactory.StartActiveRegistrar(ctx, account, nodeID, rcm.hostname, client)

	logger.Printf("Registered a connection (%s, %s)", account, nodeID)
	return nil
}

func (rcm *GatewayConnectionRegistrar) Unregister(ctx context.Context, account string, nodeID string) {
	UnregisterWithRedis(rcm.redisClient, account, nodeID, rcm.hostname)
	rcm.localConnectionRegistrar.Unregister(ctx, account, nodeID)

	rcm.activeConnectionRegistrarFactory.StopActiveRegistrar(ctx, account, nodeID)

	logger.Log.Printf("Unregistered a connection (%s, %s)", account, nodeID)
}
