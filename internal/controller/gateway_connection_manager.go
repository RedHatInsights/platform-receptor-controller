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

func (rcm *GatewayConnectionRegistrar) Register(ctx context.Context, account string, nodeID string, client Receptor) error {
	logger := logger.Log.WithFields(logrus.Fields{"account": account, "nodeID": nodeID})

	hostNameFromRedis, err := GetRedisConnection(rcm.redisClient, account, nodeID)
	if err != nil && err != redis.Nil {
		// possible transient error
		metrics.redisConnectionError.Inc()
		logger.Warn("Error getting connection from redis:", err)
		return err
	}

	if hostNameFromRedis == rcm.hostname {
		// there is already a connection for this account / node id running on this pod
		logger.Warn("Attempting to register duplicate connection on the same pod")
		metrics.duplicateConnectionCounter.Inc()
		return DuplicateConnectionError{}
	} else if hostNameFromRedis != rcm.hostname {
		logger.Debug("Host name from redis doesn't match our host name")

		if isPodRunning("cfg.GatewayClusterServiceName", hostNameFromRedis) == true {
			// the connection metadata exists and the pod is still running
			logger.Warn("Attempting to register duplicate connection on a different pod")
			metrics.duplicateConnectionCounter.Inc()
			return DuplicateConnectionError{}
		} else {
			// the connection metadata exists in redis but the pod does not exist
			metrics.unregisterStaleConnectionFromRedis.Inc()

			// the connection metadata exists but pod no longer exists
			UnregisterWithRedis(rcm.redisClient, account, nodeID, hostNameFromRedis)
		}
	}

	err = RegisterWithRedis(rcm.redisClient, account, nodeID, rcm.hostname)
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
