package main

import (
	"context"
	"flag"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	c "github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/api"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/ws"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/utils"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/gorilla/mux"
)

const (
	OPENAPI_SPEC_FILE = "/opt/app-root/src/api/api.spec.file"
)

func closeConnections(cm *c.ConnectionManager, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	connections := cm.GetAllConnections()
	for _, conn := range connections {
		for _, client := range conn {
			client.Close()
		}
	}
	time.Sleep(timeout)
}

func main() {
	var wsAddr = flag.String("wsAddr", ":8080", "Hostname:port of the websocket server")
	var mgmtAddr = flag.String("mgmtAddr", ":9090", "Hostname:port of the management server")
	flag.Parse()

	logger.InitLogger()

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

	cm := c.NewConnectionManager()
	rd := c.NewResponseReactorFactory()
	rs := c.NewReceptorServiceFactory(kw, cfg)
	md := c.NewMessageDispatcherFactory(kc)
	rc := ws.NewReceptorController(cfg, cm, wsMux, rd, md, rs)
	rc.Routes()

	apiMux := mux.NewRouter()
	apiMux.Use(request_id.ConfiguredRequestID("x-rh-insights-request-id"))

	apiSpecServer := api.NewApiSpecServer(apiMux, OPENAPI_SPEC_FILE)
	apiSpecServer.Routes()

	mgmtServer := api.NewManagementServer(cm, apiMux, cfg)
	mgmtServer.Routes()

	jr := api.NewJobReceiver(cm, apiMux, kw, cfg)
	jr.Routes()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	apiSrv := utils.StartHTTPServer(*mgmtAddr, "management", apiMux)
	wsSrv := utils.StartHTTPServer(*wsAddr, "websocket", wsMux)
	wsSrv.RegisterOnShutdown(func() { closeConnections(cm, wg, cfg.HttpShutdownTimeout) })

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signalChan
	logger.Log.Info("Received signal to shutdown: ", sig)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HttpShutdownTimeout)
	defer cancel()

	utils.ShutdownHTTPServer(ctx, "management", apiSrv)
	utils.ShutdownHTTPServer(ctx, "websocket", wsSrv)

	wg.Wait()
	logger.Log.Info("Receptor-Controller shutting down")
}
