package controller

import (
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	"github.com/go-redis/redis"
)

var hostname string = utils.GetHostname()

func addIndexes(client *redis.Client, account, nodeID, hostname string) {
	client.SAdd("connections", account+":"+nodeID+":"+hostname) // get all connections
	client.SAdd(account, nodeID+":"+hostname)                   // get all account connections
	client.SAdd(hostname, account+":"+nodeID)                   // get all pod connections
}

func removeIndexes(client *redis.Client, account, nodeID, hostname string) {
	client.SRem("connections", account+":"+nodeID+":"+hostname)
	client.SRem(account, nodeID+":"+hostname)
	client.SRem(hostname, account+":"+nodeID)
}

func ExistsInRedis(client *redis.Client, account, nodeID string) bool {
	return client.Exists(account+":"+nodeID).Val() != 0
}

func RegisterWithRedis(client *redis.Client, account, nodeID string) error {
	var res bool
	var regErr error

	_, err := client.TxPipelined(func(pipe redis.Pipeliner) error {
		res, regErr = client.SetNX(account+":"+nodeID, hostname, 0).Result()
		addIndexes(client, account, nodeID, hostname)
		return regErr
	})

	if err != nil {
		logger.Log.Print("Error attempting to register connection to Redis")
		return err
	}
	if !res {
		logger.Log.Printf("Connection (%s, %s) already found. Not registering.", account, nodeID)
		return DuplicateConnectionError{}
	}

	logger.Log.Printf("Registered a connection (%s, %s) to Redis", account, nodeID)
	return nil
}

func UnregisterWithRedis(client *redis.Client, account, nodeID string) {
	_, err := client.TxPipelined(func(pipe redis.Pipeliner) error {
		client.Del(account + ":" + nodeID)
		removeIndexes(client, account, nodeID, hostname)
		return nil
	})

	if err != nil {
		logger.Log.Print("Error attempting to unregister connection from Redis")
	}
}

func GetRedisConnection(client *redis.Client, account, nodeID string) (string, error) {
	return client.Get(account + ":" + nodeID).Result()
}

func GetRedisConnectionsByAccount(client *redis.Client, account string) ([]string, error) {
	return client.SMembers(account).Result()
}

func GetRedisConnectionsByHost(client *redis.Client, hostname string) ([]string, error) {
	return client.SMembers(hostname).Result()
}

func GetAllRedisConnections(client *redis.Client) ([]string, error) {
	return client.SMembers("connections").Result()
}
