package controller

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/go-redis/redis"

	"github.com/sirupsen/logrus"
)

type ActiveConnectionRegistrarFactory interface {
	StartActiveRegistrar(ctx context.Context, account string, nodeID string, hostname string, receptor Receptor) error
	StopActiveRegistrar(ctx context.Context, account string, nodeID string) error
}

type RedisActiveConnectionRegistrarFactory struct {
	config          *config.Config
	redisClient     *redis.Client
	hostname        string
	cancellationMap cancellationMap
}

type cancellationMap struct {
	cancelFuncs map[string]context.CancelFunc
	sync.RWMutex
}

func buildCancelMapKey(account, nodeID string) string {
	return fmt.Sprintf("%s:%s", account, nodeID)
}

func NewActiveConnectionRegistrarFactory(cfg *config.Config, rdc *redis.Client, hostname string) ActiveConnectionRegistrarFactory {
	cancelFuncsMap := cancellationMap{
		cancelFuncs: make(map[string]context.CancelFunc),
	}

	factory := RedisActiveConnectionRegistrarFactory{
		config:          cfg,
		redisClient:     rdc,
		hostname:        hostname,
		cancellationMap: cancelFuncsMap,
	}

	return &factory
}

func (f *RedisActiveConnectionRegistrarFactory) StartActiveRegistrar(ctx context.Context, account string, nodeID string, hostname string, client Receptor) error {

	logger := logger.Log.WithFields(logrus.Fields{"account": account, "nodeID": nodeID})

	ctx, cancel := context.WithCancel(ctx)

	logger.Debug("Starting ActiveConnectionRegistrar")

	go startActiveRegistrar(ctx, logger, f.config, f.redisClient, account, nodeID, f.hostname, client)

	f.cancellationMap.Lock()
	f.cancellationMap.cancelFuncs[buildCancelMapKey(account, nodeID)] = cancel
	f.cancellationMap.Unlock()

	return nil
}

func (f *RedisActiveConnectionRegistrarFactory) StopActiveRegistrar(ctx context.Context, account string, nodeID string) error {
	logger := logger.Log.WithFields(logrus.Fields{"account": account, "nodeID": nodeID})

	logger.Debug("Attempting to stop ActiveConnectionRegistrar")

	f.cancellationMap.Lock()
	cancel, exists := f.cancellationMap.cancelFuncs[buildCancelMapKey(account, nodeID)]
	f.cancellationMap.Unlock()

	if exists == false {
		logger.Debug("Unable to locate running ActiveConnectionRegistrar")
		return errors.New("unable to locate running active connection registrar")
	}

	cancel()
	logger.Debug("Cancelled running ActiveConnectionRegistrar")

	f.cancellationMap.Lock()
	delete(f.cancellationMap.cancelFuncs, buildCancelMapKey(account, nodeID))
	f.cancellationMap.Unlock()

	return nil
}

func startActiveRegistrar(ctx context.Context, logger *logrus.Entry, cfg *config.Config, redisClient *redis.Client, account string, nodeID string, hostname string, receptor Receptor) {

	ticker := time.NewTicker(cfg.GatewayActiveConnectionRegistrarPollDelay)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("Active Connection Registrar cancelled: ", ctx.Err())
			return
		case <-ticker.C:
			logger.Debug("Active Connection Registrar running")

			hostNameFromRedis, err := GetRedisConnection(redisClient, account, nodeID)
			if err != nil && err != redis.Nil {
				// possible transient error
				// FIXME: increment a redis connection error metric
				// FIXME: log it
				logger.Warn("Error getting connection from redis:", err)
				continue
			}

			logger.Debug("hostNameFromRedis:", hostNameFromRedis)

			if hostNameFromRedis == "" { // Connection is not registered
				err := registerAndCloseConnectionOnDuplicate(ctx, logger, redisClient, account, nodeID, hostname, receptor)
				if err != nil {
					// Could be a transient connection error
					logger.Warn("Unable to register connection in global connection registry")
				}
			} else if hostNameFromRedis != hostname {
				logger.Debug("Host name from redis doesn't match our host name")

				if isPodRunning(cfg.GatewayClusterServiceName, hostNameFromRedis) == true {
					// the connection metadata exists and the pod is still running
					closeConnectionDueToDuplication(ctx, logger, receptor)
					return
				} else {
					metrics.unregisterStaleConnectionFromRedis.Inc()

					// the connection metadata exists but pod no longer exists
					UnregisterWithRedis(redisClient, account, nodeID, hostNameFromRedis)

					err = registerAndCloseConnectionOnDuplicate(ctx, logger, redisClient, account, nodeID, hostname, receptor)
				}

			} else if hostNameFromRedis == hostname {
				logger.Debug("Redis connection registry entry looks correct")
			}
		}
	}
}

func registerAndCloseConnectionOnDuplicate(ctx context.Context, logger *logrus.Entry, redisClient *redis.Client, account string, nodeID string, hostname string, receptor Receptor) error {
	err := RegisterWithRedis(redisClient, account, nodeID, hostname)

	metrics.reRegisterConnectionWithRedis.Inc()

	if _, ok := err.(*DuplicateConnectionError); ok {
		closeConnectionDueToDuplication(ctx, logger, receptor)
		return err
	}

	return err
}

func closeConnectionDueToDuplication(ctx context.Context, logger *logrus.Entry, receptor Receptor) {
	// Another connection beat us to the punch...We've gotta close the connection
	logger.Warn("Another connection was created before this one...closing connection")
	metrics.autoConnectionClosureDueToDuplicateConnection.Inc()
	// FIXME:  I don't really like this...it seems dirty but I'm not sure how
	// to cleanly start closing things down from here
	receptor.Close(ctx)
}

type RunningPods map[string]bool

func getRunningPods(dnsName string) (RunningPods, error) {
	runningPodMap := make(RunningPods)

	hostnames, err := net.LookupHost(dnsName)
	if err != nil {
		logger.Log.WithFields(logrus.Fields{"error": err}).Warnf("Unable to locate running pods.  DNS entry for %s was not found", dnsName)
		metrics.podRunningStatusLookupFailure.Inc()
		return nil, err
	}

	for _, e := range hostnames {
		runningPodMap[e] = true
	}

	return runningPodMap, nil
}

func isPodRunning(headlessDNSName string, podName string) bool {
	runningPods, _ := getRunningPods(headlessDNSName)
	// FIXME: err ^^

	_, exists := runningPods[podName]

	return exists
}
