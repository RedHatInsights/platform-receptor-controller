package controller

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	"github.com/go-redis/redis"
)

type RedisConnector interface {
	Exists(account, nodeID string) bool
	Register(account, nodeID string) error
	Unregister(account, nodeID string)
}

type redisConnection struct {
	client   *redis.Client
	cfg      *config.Config
	hostname string
}

func NewRedisConnector(cfg *config.Config) RedisConnector {
	return &redisConnection{
		client: redis.NewClient(&redis.Options{
			Addr:     (cfg.RedisHost + ":" + cfg.RedisPort),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}),
		cfg:      cfg,
		hostname: utils.GetHostname(),
	}
}

func (rc *redisConnection) updateIndexes(account, nodeID, hostname string) {
	rc.client.SAdd("connections", account+":"+nodeID+":"+rc.hostname) // get all connections
	rc.client.SAdd(account, nodeID+":"+rc.hostname)                   // get all account connections
	rc.client.SAdd(rc.hostname, account+":"+nodeID)                   // get all pod connections
}

func (rc *redisConnection) Exists(account, nodeID string) bool {
	return rc.client.Exists(account+":"+nodeID).Val() != 0
}

func (rc *redisConnection) Register(account, nodeID string) error {
	res, err := rc.client.SetNX(account+":"+nodeID, rc.hostname, 0).Result()

	if err != nil {
		logger.Log.Warn("Error attempting to register connection")
		return err
	}
	if !res {
		logger.Log.Printf("Connection (%s, %s) already found. Not registering.")
		return nil
	}

	rc.updateIndexes(account, nodeID, rc.hostname)
	logger.Log.Printf("Registered a connection (%s, %s)", account, nodeID)
	return nil
}

func (rc *redisConnection) Unregister(account, nodeID string) {
	if res := rc.client.Del(account + ":" + nodeID).Val(); res != 1 {
		logger.Log.Warn("Attempting to unregister a connection that does not exist")
		return
	}
	logger.Log.Printf("Unregistered a connection (%s, %s)", account, nodeID)
}

func (rc *redisConnection) GetConnection(account, nodeID string) string {
	return rc.client.Get(account + ":" + nodeID).Val()
}

func (rc *redisConnection) GetConnectionsByAccount(account string) []string {
	return rc.client.SMembers(account).Val()
}

func (rc *redisConnection) GetConnectionsByHost(hostname string) []string {
	return rc.client.SMembers(hostname).Val()
}

func (rc *redisConnection) GetAllConnections() []string {
	return rc.client.SMembers("connections").Val()
}
