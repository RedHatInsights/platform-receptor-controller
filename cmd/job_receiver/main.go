package main

import (
	"flag"
	"fmt"
	"log"
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

func main() {
	var mgmtAddr = flag.String("mgmtAddr", ":8081", "Hostname:port of the management server")
	flag.Parse()

	logger.InitLogger()

	logger.Log.Info("Starting Receptor-Controller Job-Receiver service")

	cfg := config.GetConfig()
	logger.Log.Info("Receptor Controller configuration:\n", cfg)

	redisClient, err := initRedis(cfg)
	if err != nil {
		log.Fatal("Unable to connect to Redis:", err)
	}

	var connectionLocator controller.ConnectionLocator
	connectionLocator = &api.RedisConnectionLocator{Client: redisClient, Cfg: cfg}
	mgmtMux := mux.NewRouter()
	mgmtMux.Use(request_id.ConfiguredRequestID("x-rh-insights-request-id"))
	mgmtServer := api.NewManagementServer(connectionLocator, mgmtMux, cfg)
	mgmtServer.Routes()

	kw := queue.StartProducer(&queue.ProducerConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaResponsesTopic,
	})

	jr := api.NewJobReceiver(connectionLocator, mgmtMux, kw, cfg)
	jr.Routes()

	go func() {
		log.Println("Starting management web server on", *mgmtAddr)
		if err := http.ListenAndServe(*mgmtAddr, mgmtMux); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Blocking waiting for signal")
	<-signalChan
}
