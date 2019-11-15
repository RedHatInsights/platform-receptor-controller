package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	var ws_addr = flag.String("ws_addr", ":8080", "Hostname:port of the websocket server")
	var mgmt_addr = flag.String("mgmt_addr", ":9090", "Hostname:port of the management server")
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
		// FIXME:  If this fails (port already in use), the error will be ignored
		log.Println("Starting management web server on", *mgmt_addr)
		if err := http.ListenAndServe(*mgmt_addr, mgmt_mux); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	log.Println("Starting websocket server on", *ws_addr)
	if err := http.ListenAndServe(*ws_addr, ws_mux); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}
