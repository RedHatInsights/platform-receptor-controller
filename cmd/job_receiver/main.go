package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/api"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     (cfg.RedisHost + ":" + cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	pong, err := client.Ping().Result()
	fmt.Println(pong, err)
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
	var mgmtAddr = flag.String("mgmtAddr", ":8081", "Hostname:port of the management server")
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

	apiMux.Handle("/metrics", promhttp.Handler())

	mgmtServer := api.NewManagementServer(connectionLocator, apiMux, cfg)
	mgmtServer.Routes()

	kw := queue.StartProducer(&queue.ProducerConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaResponsesTopic,
	})

	jr := api.NewJobReceiver(connectionLocator, apiMux, kw, cfg)
	jr.Routes()

	go func() {
		logger.Log.Println("Starting management web server on", *mgmtAddr)
		if err := http.ListenAndServe(*mgmtAddr, apiMux); err != nil {
			logger.Log.Fatal("ListenAndServe:", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Log.Println("Blocking waiting for signal")
	<-signalChan
}
