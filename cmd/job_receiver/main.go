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
	"github.com/gorilla/mux"
)

func main() {
	var mgmtAddr = flag.String("mgmtAddr", ":8081", "Hostname:port of the management server")
	flag.Parse()

	wsConfig := ws.GetWebSocketConfig()

	cm := c.NewConnectionManager()
	mgmtMux := mux.NewRouter()
	mgmtServer := api.NewManagementServer(cm, mgmtMux, wsConfig.ServiceToServiceCredentials)
	mgmtServer.Routes()

	kw := queue.StartProducer(queue.GetProducer())

	jr := api.NewJobReceiver(cm, mgmtMux, kw, wsConfig.ServiceToServiceCredentials)
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
