package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	c "github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/api"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/ws"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

const (
	OPENAPI_SPEC_FILE = "/opt/app-root/src/api/api.spec.file"
)

func main() {
	var wsAddr = flag.String("wsAddr", ":8080", "Hostname:port of the websocket server")
	var mgmtAddr = flag.String("mgmtAddr", ":9090", "Hostname:port of the management server")
	flag.Parse()

	logger.InitLogger()

	logger.Log.Info("Starting Receptor-Controller service")

	wsConfig := ws.GetWebSocketConfig()
	logger.Log.Info("WebSocket configuration:\n", wsConfig)

	wsMux := mux.NewRouter()
	wsMux.Use(request_id.ConfiguredRequestID("x-rh-insights-request-id"))

	cm := c.NewConnectionManager()
	kw := queue.StartProducer(queue.GetProducer())
	kc := queue.GetConsumer()
	rd := c.NewResponseReactorFactory()
	rs := c.NewReceptorServiceFactory(kw)
	md := c.NewMessageDispatcherFactory(kc)
	rc := ws.NewReceptorController(wsConfig, cm, wsMux, rd, md, rs)
	rc.Routes()

	apiMux := mux.NewRouter()
	apiMux.Use(request_id.ConfiguredRequestID("x-rh-insights-request-id"))

	apiSpecServer := api.NewApiSpecServer(apiMux, OPENAPI_SPEC_FILE)
	apiSpecServer.Routes()

	mgmtServer := api.NewManagementServer(cm, apiMux, wsConfig.ServiceToServiceCredentials)
	mgmtServer.Routes()

	jr := api.NewJobReceiver(cm, apiMux, kw, wsConfig.ServiceToServiceCredentials)
	jr.Routes()

	apiMux.Handle("/metrics", promhttp.Handler())

	go func() {
		logger.Log.Info("Starting management web server:  ", *mgmtAddr)
		if err := http.ListenAndServe(*mgmtAddr, handlers.LoggingHandler(os.Stdout, apiMux)); err != nil {
			logger.Log.WithFields(logrus.Fields{"error": err}).Fatal("managment web server error")
		}
	}()

	go func() {
		logger.Log.Info("Starting websocket server on:  ", *wsAddr)
		if err := http.ListenAndServe(*wsAddr, handlers.LoggingHandler(os.Stdout, wsMux)); err != nil {
			logger.Log.WithFields(logrus.Fields{"error": err}).Fatal("websocket server error")
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan
	logger.Log.Debug("Receptor-Controller shutting down")
}
