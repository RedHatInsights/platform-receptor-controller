package controller

import (
//	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
//	"github.com/sirupsen/logrus"
)

type RedisConnectionLocator struct {
}

func (cm *RedisConnectionLocator) GetConnection(account string, node_id string) Receptor {
	var conn Receptor
	return conn
}

func (cm *RedisConnectionLocator) GetConnectionsByAccount(account string) map[string]Receptor {
	connectionsPerAccount := make(map[string]Receptor)

	return connectionsPerAccount
}

func (cm *RedisConnectionLocator) GetAllConnections() map[string]map[string]Receptor {
	connectionMap := make(map[string]map[string]Receptor)

	return connectionMap
}
