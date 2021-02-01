package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/api"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     (cfg.RedisHost + ":" + cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func verifyConfiguration(cfg *config.Config) error {
	if cfg.JobReceiverReceptorProxyClientID == "" {
		return errors.New(config.JOB_RECEIVER_RECEPTOR_PROXY_CLIENT_ID + " configuration missing")
	}

	if cfg.JobReceiverReceptorProxyPSK == "" {
		return errors.New(config.JOB_RECEIVER_RECEPTOR_PROXY_PSK + " configuration missing")
	}

	return nil
}

func main() {
	defaultMgmtAddr := ":8081"

	if clowder.IsClowderEnabled() {
		defaultMgmtAddr = fmt.Sprintf(":%d", *clowder.LoadedConfig.PrivatePort)
	}

	var mgmtAddr = flag.String("mgmtAddr", defaultMgmtAddr, "Hostname:port of the management server")
	flag.Parse()

	logger.InitLogger()

	logger.Log.Info("Starting Receptor-Controller Job-Receiver service")

	cfg := config.GetConfig()
	logger.Log.Info("Receptor Controller configuration:\n", cfg)

	err := verifyConfiguration(cfg)
	if err != nil {
		logger.Log.Fatal("Configuration error encountered during startup: ", err)
	}

	redisClient, err := initRedis(cfg)
	if err != nil {
		logger.Log.Fatal("Unable to connect to Redis: ", err)
	}

	var connectionLocator controller.ConnectionLocator
	connectionLocator = &api.RedisConnectionLocator{Client: redisClient, Cfg: cfg}

	apiMux := mux.NewRouter()
	apiMux.Use(request_id.ConfiguredRequestID("x-rh-insights-request-id"))

	monitoringServer := api.NewMonitoringServer(apiMux, cfg)
	monitoringServer.Routes()

	mgmtServer := api.NewManagementServer(connectionLocator, apiMux, cfg)
	mgmtServer.Routes()

	jr := api.NewJobReceiver(connectionLocator, apiMux, cfg)
	jr.Routes()

	apiSrv := utils.StartHTTPServer(*mgmtAddr, "management", apiMux)

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signalChan
	logger.Log.Info("Received signal to shutdown: ", sig)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HttpShutdownTimeout)
	defer cancel()

	utils.ShutdownHTTPServer(ctx, "management", apiSrv)

	logger.Log.Info("Receptor-Controller shutting down")
}
