package controller

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	"github.com/go-redis/redis"
)

type RedisInterface interface {
	Exists(account, node_id string) bool
	Register(account, node_id string) error
	Unregister(account, node_id string)
}

type RedisManager struct {
	client   *redis.Client
	cfg      *config.Config
	hostname string
}

func NewRedisManager(cfg *config.Config) RedisInterface {
	return &RedisManager{
		client: redis.NewClient(&redis.Options{
			Addr:     (cfg.RedisHost + ":" + cfg.RedisPort),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}),
		cfg:      cfg,
		hostname: utils.GetHostname(),
	}
}

func (rm *RedisManager) Exists(account, node_id string) bool {
	if rm.client.Exists(account+":"+node_id).Val() != 0 {
		return true
	}
	return false
}

func (rm *RedisManager) Register(account, node_id string) error {
	if err := rm.client.Set(account+":"+node_id, rm.hostname, 0).Err(); err != nil {
		logger.Log.Warn("Unable to register connection")
		return err
	}
	logger.Log.Printf("Registered a connection (%s, %s)", account, node_id)
	return nil
}

func (rm *RedisManager) Unregister(account, node_id string) {
	if res := rm.client.Del(account + ":" + node_id).Val(); res != 1 {
		logger.Log.Warn("Attempting to unregister a connection that does not exist")
		return
	}
	logger.Log.Printf("Unregistered a connection (%s, %s)", account, node_id)
}

func (rm *RedisManager) GetConnection(account, node_id string) string { // not implemented
	var conn string
	return conn
}

func (rm *RedisManager) GetConnectionsByAccount(account string) []string { // not implemented
	var connectionsPerAccount []string
	return connectionsPerAccount
}

func (rm *RedisManager) GetAllConnections() []string { // not implemented
	var connections []string
	return connections
}
