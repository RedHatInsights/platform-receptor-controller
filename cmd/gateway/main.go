package main

import (
	"context"
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	c "github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/api"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/ws"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/gorilla/mux"
)

const (
	OPENAPI_SPEC_FILE = "/opt/app-root/src/api/api.spec.file"
)

func closeConnections(cm c.ConnectionLocator, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	ctx := context.TODO()
	connections := cm.GetAllConnections(ctx)
	for _, conn := range connections {
		for _, client := range conn {
			client.Close(ctx)
		}
	}
	time.Sleep(timeout)
}

func configureConnectionRegistrar(cfg *config.Config, localCM c.ConnectionRegistrar) c.ConnectionRegistrar {
	switch strings.ToLower(cfg.GatewayConnectionRegistrarImpl) {
	case "redis":
		logger.Log.Info("Using GatewayConnectionRegistrar as the ConnectionRegistrar impl." +
			"  Connections will be registered with Redis.")

		redisClient := redis.NewClient(&redis.Options{
			Addr:     (cfg.RedisHost + ":" + cfg.RedisPort),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		_, err := redisClient.Ping().Result()
		if err != nil {
			logger.Log.Fatal("Unable to connect to redis: ", err)
		}

		ipAddr := utils.GetIPAddress()
		if ipAddr == nil {
			logger.Log.Fatal("Unable to determine IP address")
		}

		activeConnectionRegistrarFactory := c.NewActiveConnectionRegistrarFactory(cfg, redisClient, ipAddr.String())

		return c.NewGatewayConnectionRegistrar(redisClient, localCM, activeConnectionRegistrarFactory, ipAddr.String())
	case "local":
		logger.Log.Info("Using LocalConnectionManager as the ConnectionRegistrar impl." +
			"  Connections will NOT be registered with Redis.")

		return localCM
	default:
		logger.Log.Fatalf("Invalid configuration value for %s!", config.GATEWAY_CONNECTION_REGISTRAR_IMPL)
		return nil
	}
}

func main() {
	logger.InitLogger()

	var wsAddrPtr = flag.String("wsAddr", ":8080", "Hostname:port of the websocket server")
	var mgmtAddrPtr = flag.String("mgmtAddr", ":9090", "Hostname:port of the management server")
	var monitoringAddrPtr = flag.String("monitoringAddr", ":10000", "Hostname:port of the monitoring server")
	flag.Parse()

	var wsAddr = *wsAddrPtr
	var mgmtAddr = *mgmtAddrPtr
	var monitoringAddr = *monitoringAddrPtr

	if clowder.IsClowderEnabled() {
		logger.Log.Info("Receptor-Controller is running in a Clowderized environment...overriding port configuration!!")
		wsAddr = fmt.Sprintf(":%d", *clowder.LoadedConfig.PublicPort)
		mgmtAddr = fmt.Sprintf(":%d", *clowder.LoadedConfig.PrivatePort)
		monitoringAddr = fmt.Sprintf(":%d", clowder.LoadedConfig.MetricsPort)
	}

	logger.Log.Info("Starting Receptor-Controller service")

	cfg := config.GetConfig()
	logger.Log.Info("Receptor Controller configuration:\n", cfg)

	wsMux := mux.NewRouter()
	wsMux.Use(request_id.ConfiguredRequestID("x-rh-insights-request-id"))

	kw := queue.StartProducer(&queue.ProducerConfig{
		Brokers:    cfg.KafkaBrokers,
		Topic:      cfg.KafkaResponsesTopic,
		BatchSize:  cfg.KafkaResponsesBatchSize,
		BatchBytes: cfg.KafkaResponsesBatchBytes,
	})

	kc := &queue.ConsumerConfig{
		Brokers:        cfg.KafkaBrokers,
		Topic:          cfg.KafkaJobsTopic,
		GroupID:        cfg.KafkaGroupID,
		ConsumerOffset: cfg.KafkaConsumerOffset,
	}

	var gatewayCR c.ConnectionRegistrar

	localCM := c.NewLocalConnectionManager()
	gatewayCR = configureConnectionRegistrar(cfg, localCM)

	rd := c.NewResponseReactorFactory()
	rs := c.NewReceptorServiceFactory(kw, cfg)
	md := c.NewMessageDispatcherFactory(kc)
	rc := ws.NewReceptorController(cfg, gatewayCR, wsMux, rd, md, rs)
	rc.Routes()

	apiMux := mux.NewRouter()
	apiMux.Use(request_id.ConfiguredRequestID("x-rh-insights-request-id"))

	apiSpecServer := api.NewApiSpecServer(apiMux, OPENAPI_SPEC_FILE)
	apiSpecServer.Routes()

	mgmtServer := api.NewManagementServer(localCM, apiMux, cfg)
	mgmtServer.Routes()

	jr := api.NewJobReceiver(localCM, apiMux, cfg)
	jr.Routes()

	monitoringMux := mux.NewRouter()
	monitoringMux.Use(request_id.ConfiguredRequestID("x-rh-insights-request-id"))

	monitoringServer := api.NewMonitoringServer(monitoringMux, cfg)
	monitoringServer.Routes()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	monitoringSrv := utils.StartHTTPServer(monitoringAddr, "monitoring", monitoringMux)
	apiSrv := utils.StartHTTPServer(mgmtAddr, "management", apiMux)
	wsSrv := utils.StartHTTPServer(wsAddr, "websocket", wsMux)
	wsSrv.RegisterOnShutdown(func() { closeConnections(localCM, wg, cfg.HttpShutdownTimeout) })

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signalChan
	logger.Log.Info("Received signal to shutdown: ", sig)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HttpShutdownTimeout)
	defer cancel()

	utils.ShutdownHTTPServer(ctx, "monitoring", monitoringSrv)
	utils.ShutdownHTTPServer(ctx, "management", apiSrv)
	utils.ShutdownHTTPServer(ctx, "websocket", wsSrv)

	wg.Wait()
	logger.Log.Info("Receptor-Controller shutting down")
}
