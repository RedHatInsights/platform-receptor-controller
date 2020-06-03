package api

import (
	"fmt"
	//"os"
	"strings"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	//"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/go-redis/redis"
)

type RedisConnectionLocator struct {
	RedisConnection *redis.Client
}

func (rcl *RedisConnectionLocator) GetConnection(account string, node_id string) controller.Receptor {
	var conn controller.Receptor

	/*
		var url string
		url = os.Getenv("GATEWAY_URL")
		if len(url) == 0 {
			logger.Log.Printf("GATEWAY_URL env var is not set\n")
		}
		logger.Log.Printf("GATEWAY_URL: %s\n", url)
	*/

	direct_lookup_key := fmt.Sprintf("%s:%s", account, node_id)

	var podName string
	var err error

	if podName, err = rcl.RedisConnection.Get(direct_lookup_key).Result(); err != nil {
		// FIXME: log error, return an error
		return nil
	}
	fmt.Println("get by account/nodeid result:", podName)
	fmt.Printf("get by account/nodeid result (type):%T\n", podName)
	fmt.Println("get by account/nodeid err:", err)

	if podName == "" {
		return nil
	}

	url := fmt.Sprintf("http://%s:9090", podName)

	conn = &ReceptorHttpProxy{Url: url, AccountNumber: account, NodeID: node_id}

	return conn
}

func (rcl *RedisConnectionLocator) GetConnectionsByAccount(account string) map[string]controller.Receptor {

	var cursor uint64
	var err error

	connectionsPerAccount := make(map[string]controller.Receptor)

	connections := make(map[string]interface{})

	for {
		var keys []string
		pattern := fmt.Sprintf("%s:*", account)
		if keys, cursor, err = rcl.RedisConnection.Scan(cursor, pattern, 50).Result(); err != nil {
			// FIXME: log error, return an error
			return connectionsPerAccount
		}

		if len(keys) > 0 {
			fmt.Printf("found %d keys\n", len(keys))
		}

		for _, key := range keys {
			connections[key] = nil
		}

		if cursor == 0 {
			break
		}
	}

	for key, _ := range connections {
		s := strings.Split(key, ":")
		proxy := rcl.GetConnection(s[0], s[1])
		connectionsPerAccount[key] = proxy
	}

	return connectionsPerAccount
}

func (rcl *RedisConnectionLocator) GetAllConnections() map[string]map[string]controller.Receptor {

	var cursor uint64
	var err error

	connectionMap := make(map[string]map[string]controller.Receptor)

	connections := make(map[string]interface{})

	for {
		var keys []string
		if keys, cursor, err = rcl.RedisConnection.Scan(cursor, "*", 50).Result(); err != nil {
			// FIXME: log error, return an error
			return connectionMap
		}

		if len(keys) > 0 {
			fmt.Printf("found %d keys\n", len(keys))
		}

		for _, key := range keys {
			connections[key] = nil
		}

		if cursor == 0 {
			break
		}
	}

	for key, _ := range connections {
		s := strings.Split(key, ":")
		account := s[0]
		nodeId := s[1]
		proxy := rcl.GetConnection(account, nodeId)
		_, exists := connectionMap[account]
		if exists == false {
			connectionMap[account] = make(map[string]controller.Receptor)
		}
		connectionMap[account][nodeId] = proxy
	}

	return connectionMap
}
