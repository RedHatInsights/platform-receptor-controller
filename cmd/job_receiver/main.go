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

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
)

func main() {
	var mgmtAddr = flag.String("mgmtAddr", ":8081", "Hostname:port of the management server")
	flag.Parse()

	cm := c.NewConnectionManager()
	mgmtMux := http.NewServeMux()
	mgmtServer := c.NewManagementServer(cm, mgmtMux)
	mgmtServer.Routes()

	kw := queue.StartProducer(queue.Get())

	jr := c.NewJobReceiver(cm, mgmtMux, kw)
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
