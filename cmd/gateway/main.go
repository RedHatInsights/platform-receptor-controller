package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	c "github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/api"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/ws"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/redhatinsights/platform-go-middlewares/identity"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	var wsAddr = flag.String("wsAddr", ":8080", "Hostname:port of the websocket server")
	var mgmtAddr = flag.String("mgmtAddr", ":9090", "Hostname:port of the management server")
	flag.Parse()

	wsConfig := ws.GetWebSocketConfig()
	log.Println("WebSocket configuration:")
	log.Println(wsConfig)

	wsMux := mux.NewRouter()
	cm := c.NewConnectionManager()
	kw := queue.StartProducer(queue.GetProducer())
	d := c.NewResponseDispatcherFactory(kw)
	rc := ws.NewReceptorController(wsConfig, cm, wsMux, d)
	rc.Routes()

	apiMux := mux.NewRouter()

	apiSpecServer := api.NewApiSpecServer(apiMux)
	apiSpecServer.Routes()

	securedApiMux := apiMux.PathPrefix("/").Subrouter()
	securedApiMux.Use(identity.EnforceIdentity)

	mgmtServer := api.NewManagementServer(cm, securedApiMux)
	mgmtServer.Routes()

	jr := api.NewJobReceiver(cm, securedApiMux, kw)
	jr.Routes()

	go func() {
		log.Println("Starting management web server on", *mgmtAddr)
		if err := http.ListenAndServe(*mgmtAddr, handlers.LoggingHandler(os.Stdout, apiMux)); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	go func() {
		log.Println("Starting websocket server on", *wsAddr)
		if err := http.ListenAndServe(*wsAddr, handlers.LoggingHandler(os.Stdout, wsMux)); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Blocking waiting for signal")
	<-signalChan
}
