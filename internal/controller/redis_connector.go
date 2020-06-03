package controller

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	"github.com/go-redis/redis"
)

type RedisManager interface {
	Exists(account, nodeID string) bool
	Register(account, nodeID string) error
	Unregister(account, nodeID string)
}

type RedisLocator interface {
	Lookup(index string) ([]string, error)
	GetConnection(account, node_id string) (string, error)
}

func NewRedisManager(client *redis.Client) RedisManager {
	return &manager{
		client:   client,
		hostname: utils.GetHostname(),
	}
}

func NewRedisLocator(client *redis.Client) RedisLocator {
	return &locator{client: client}
}

type manager struct {
	client   *redis.Client
	hostname string
}

type locator struct {
	client *redis.Client
}

func (m *manager) updateIndexes(account, nodeID, hostname string) {
	m.client.SAdd("connections", account+":"+nodeID+":"+m.hostname) // get all connections
	m.client.SAdd(account, nodeID+":"+m.hostname)                   // get all account connections
	m.client.SAdd(m.hostname, account+":"+nodeID)                   // get all pod connections
}

func (m *manager) Exists(account, nodeID string) bool {
	return m.client.Exists(account+":"+nodeID).Val() != 0
}

func (m *manager) Register(account, nodeID string) error {
	res, err := m.client.SetNX(account+":"+nodeID, m.hostname, 0).Result()

	if err != nil {
		logger.Log.Warn("Error attempting to register connection")
		return err
	}
	if !res {
		logger.Log.Warnf("Connection (%s, %s) already found. Not registering.", account, nodeID)
		return nil
	}

	m.updateIndexes(account, nodeID, m.hostname)
	logger.Log.Printf("Registered a connection (%s, %s)", account, nodeID)
	return nil
}

func (m *manager) Unregister(account, nodeID string) {
	if res := m.client.Del(account + ":" + nodeID).Val(); res != 1 {
		logger.Log.Warn("Attempting to unregister a connection that does not exist")
		return
	}
	logger.Log.Printf("Unregistered a connection (%s, %s)", account, nodeID)
}

func (l *locator) GetConnection(account, nodeID string) (string, error) {
	return l.client.Get(account + ":" + nodeID).Result()
}

func (l *locator) Lookup(index string) ([]string, error) {
	return l.client.SMembers(index).Result()
}

// func (rc *redisConnection) GetConnectionsByAccount(account string) []string {
// 	return rc.client.SMembers(account).Val()
// }

// func (rc *redisConnection) GetConnectionsByHost(hostname string) []string {
// 	return rc.client.SMembers(hostname).Val()
// }

// func (rc *redisConnection) GetAllConnections() []string {
// 	return rc.client.SMembers("connections").Val()
// }
