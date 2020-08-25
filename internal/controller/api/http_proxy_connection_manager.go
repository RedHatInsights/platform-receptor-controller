package api

import (
	"context"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

type RedisConnectionLocator struct {
	Client *redis.Client
	Cfg    *config.Config
}

func (rcl *RedisConnectionLocator) GetConnection(ctx context.Context, account string, node_id string) controller.Receptor {
	var conn controller.Receptor
	var podName string
	var err error

	log := logger.Log.WithFields(logrus.Fields{"account": account, "node_id": node_id})

	if podName, err = controller.GetRedisConnection(rcl.Client, account, node_id); err != nil {
		log.WithFields(logrus.Fields{"error": err}).Error("Error during connection lookup for account ", account)
		return nil
	}

	if podName == "" {
		log.Error("Redis lookup returned empty podname")
		return nil
	}

	conn = &ReceptorHttpProxy{
		Hostname:      podName,
		AccountNumber: account,
		NodeID:        node_id,
		Config:        rcl.Cfg,
	}

	return conn
}

func (rcl *RedisConnectionLocator) GetConnectionsByAccount(ctx context.Context, account string) map[string]controller.Receptor {

	log := logger.Log.WithFields(logrus.Fields{"account": account})

	connectionsPerAccount := make(map[string]controller.Receptor)

	accountConnections, err := controller.GetRedisConnectionsByAccount(rcl.Client, account)
	if err != nil {
		log.WithFields(logrus.Fields{"error": err}).Error("Error during connection lookup for account ", account)
		return nil
	}

	for nodeID, _ := range accountConnections {
		proxy := rcl.GetConnection(ctx, account, nodeID)
		connectionsPerAccount[nodeID] = proxy
	}

	return connectionsPerAccount
}

func (rcl *RedisConnectionLocator) GetAllConnections(ctx context.Context) map[string]map[string]controller.Receptor {

	connectionMap := make(map[string]map[string]controller.Receptor)

	connections, err := controller.GetAllRedisConnections(rcl.Client)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"error": err}).Error("Error during connection lookup for all connections")
		return nil
	}

	for account, conn := range connections {
		if _, exists := connectionMap[account]; !exists {
			connectionMap[account] = make(map[string]controller.Receptor)
		}
		for node, _ := range conn {
			proxy := rcl.GetConnection(ctx, account, node)
			connectionMap[account][node] = proxy
		}
	}

	return connectionMap
}
