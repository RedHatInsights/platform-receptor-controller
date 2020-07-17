package main

import (
	"flag"
	"fmt"

	"github.com/go-redis/redis"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
)

func init() {
	logger.InitLogger()
}

func main() {
	var action = flag.String("action", "register", "register/unregister/list")
	var accountNumber = flag.String("account", "", "Account number")
	var nodeID = flag.String("node-id", "", "Node ID")
	var ipAddr = flag.String("ip", "", "ipAddr")
	flag.Parse()

	cfg := config.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     (cfg.RedisHost + ":" + cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	switch *action {
	case "register":
		if *accountNumber == "" || *nodeID == "" || *ipAddr == "" {
			logger.Log.Fatal("Required parameters: account, node-id, ip")
		}
		controller.RegisterWithRedis(redisClient, *accountNumber, *nodeID, *ipAddr)
	case "unregister":
		if *accountNumber == "" || *nodeID == "" || *ipAddr == "" {
			logger.Log.Fatal("Required parameters: account, node-id, ip")
		}
		controller.UnregisterWithRedis(redisClient, *accountNumber, *nodeID, *ipAddr)
	case "list":
		if *accountNumber == "" || *nodeID == "" {
			logger.Log.Fatal("Required parameters: account, node-id")
		}
		fmt.Println(controller.GetRedisConnection(redisClient, *accountNumber, *nodeID))
	default:
		logger.Log.Info("Invalid action!")
	}
}
