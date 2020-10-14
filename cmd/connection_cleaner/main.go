package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
)

type RunningPods map[string]bool

type Metrics struct {
	unregisterStaleConnectionFromRedis prometheus.Counter
}

func getRunningPods(dnsName string) (RunningPods, error) {
	runningPodMap := make(map[string]bool)

	//hostnames := []string{"192.168.1.34", "192.168.2.43", "10.188.249.243"}

	fmt.Println("Looking up IP addresses for pod ", dnsName)
	hostnames, err := net.LookupHost(dnsName)
	if err != nil {
		fmt.Println("Unable to locate running pods")
		return nil, err
	}

	for _, e := range hostnames {
		runningPodMap[e] = true
	}

	return runningPodMap, nil
}

func processConnection(dryRun bool, runningPods RunningPods, redisClient *redis.Client, account, nodeID, podName string, connCount *Metrics) {
	if _, exists := runningPods[podName]; !exists {
		fmt.Printf("Pod (%s) down!  This entry should be removed:  %s:%s\n", podName, account, nodeID)
		if dryRun == false {
			controller.UnregisterWithRedis(redisClient, account, nodeID, podName)
			connCount.unregisterStaleConnectionFromRedis.Inc()
		}
	}
}

func pushValues(cfg *config.Config, connCount *Metrics) {
	if err := push.New(cfg.PromPushGW, "receptor_controller_stale_job").
		Collector(connCount.unregisterStaleConnectionFromRedis).
		Push(); err != nil {
		fmt.Println("Unable to push metrics to prometheus gateway")
	}

}

func main() {
	var podName = flag.String("pod-name", "receptor-gateway-internal", "DNS name of the internal gateway pod")
	var dryRun = flag.Bool("dry-run", false, "Just report the stale connections.  No changes will be made.")

	connCount := &Metrics{
		unregisterStaleConnectionFromRedis: promauto.NewCounter(prometheus.CounterOpts{
			Name: "receptor_controller_unregister_stale_connection_from_redis_count",
			Help: "The number of times a stale connection has been unregistered from redis",
		}),
	}

	flag.Parse()

	logger.InitLogger()

	fmt.Println("Starting Receptor-Controller Connection Cleaner")

	cfg := config.GetConfig()
	fmt.Println("Receptor Controller configuration:\n", cfg)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     (cfg.RedisHost + ":" + cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	runningPods, err := getRunningPods(*podName)
	if err != nil {
		fmt.Println("Unable to locate running pods")
		return
	}

	fmt.Println("Running pods: ", runningPods)

	allConnections, err := controller.GetAllRedisConnections(redisClient)
	if err != nil {
		fmt.Println("Error getting connection list from redis:", err)
		return
	}

	fmt.Printf("Processing %d connections\n", len(allConnections))

	for account, value := range allConnections {
		fmt.Printf("account:%s\n", account)
		fmt.Printf("value:%s\n", value)
		fmt.Printf("value:%T\n", value)
		for nodeID, podName := range value {
			processConnection(*dryRun, runningPods, redisClient, account, nodeID, podName, connCount)
		}
	}
	pushValues(cfg, connCount)
}
