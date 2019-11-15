package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	var addr = flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse()

	ws_mux := http.NewServeMux()
	cm := newConnectionManager()
	rc := newReceptorController(cm, ws_mux)
	rc.routes()

	mgmt_mux := http.NewServeMux()
	mgmt_server := newManagementServer(cm, mgmt_mux)
	mgmt_server.routes()

	jr := newJobReceiver(cm, mgmt_mux)
	jr.routes()

	go func() {
		mgmt_port := ":9090"
		log.Println("Starting management web server on", mgmt_port)
		http.ListenAndServe(mgmt_port, mgmt_mux)
	}()

	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, ws_mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}
