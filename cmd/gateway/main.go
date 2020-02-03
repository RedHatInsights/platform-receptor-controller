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
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/ws"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	var wsAddr = flag.String("wsAddr", ":8080", "Hostname:port of the websocket server")
	var mgmtAddr = flag.String("mgmtAddr", ":9090", "Hostname:port of the management server")
	flag.Parse()

	wsMux := mux.NewRouter()
	cm := c.NewConnectionManager()
	kw := queue.StartProducer(queue.GetProducer())
	d := c.NewResponseDispatcherFactory(kw)
	rc := ws.NewReceptorController(cm, wsMux, kw, d)
	rc.Routes()

	mgmtMux := mux.NewRouter()
	mgmtServer := c.NewManagementServer(cm, mgmtMux)
	mgmtServer.Routes()

	jr := c.NewJobReceiver(cm, mgmtMux, kw)
	jr.Routes()

	go func() {
		log.Println("Starting management web server on", *mgmtAddr)
		if err := http.ListenAndServe(*mgmtAddr, handlers.LoggingHandler(os.Stdout, mgmtMux)); err != nil {
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
