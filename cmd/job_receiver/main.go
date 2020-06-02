package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	c "github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller/api"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/gorilla/mux"
)

func main() {
	var mgmtAddr = flag.String("mgmtAddr", ":8081", "Hostname:port of the management server")
	flag.Parse()

	cfg := config.GetConfig()

	var cm c.ConnectionLocator
	cm = &c.RedisConnectionLocator{}
	mgmtMux := mux.NewRouter()
	mgmtServer := api.NewManagementServer(cm, mgmtMux, cfg)
	mgmtServer.Routes()

	kw := queue.StartProducer(&queue.ProducerConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaResponsesTopic,
	})

	jr := api.NewJobReceiver(cm, mgmtMux, kw, cfg)
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
