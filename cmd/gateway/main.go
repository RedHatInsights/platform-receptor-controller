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
)

func main() {
	var wsAddr = flag.String("wsAddr", ":8080", "Hostname:port of the websocket server")
	var mgmtAddr = flag.String("mgmtAddr", ":9090", "Hostname:port of the management server")
	flag.Parse()

	wsMux := http.NewServeMux()
	cm := c.NewConnectionManager()
	rc := c.NewReceptorController(cm, wsMux)
	rc.Routes()

	mgmtMux := http.NewServeMux()
	mgmtServer := c.NewManagementServer(cm, mgmtMux)
	mgmtServer.Routes()

	go func() {
		log.Println("Starting management web server on", *mgmtAddr)
		if err := http.ListenAndServe(*mgmtAddr, mgmtMux); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	go func() {
		log.Println("Starting websocket server on", *wsAddr)
		if err := http.ListenAndServe(*wsAddr, wsMux); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Blocking waiting for signal")
	<-signalChan
}
