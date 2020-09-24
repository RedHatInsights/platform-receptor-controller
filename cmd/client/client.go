package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	//"strings"
	"syscall"
	//"time"
	//"encoding/base64"

	"github.com/project-receptor/receptor/pkg/backends"
	"github.com/project-receptor/receptor/pkg/netceptor"
)

var targetUrl = flag.String("url", "ws://localhost:8080/wss/receptor-controller/gateway", "http service address")

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var headerFlags arrayFlags

var identity = `{ "identity": {"account_number": "01", "type": "User", "internal": { "org_id": "1979710" } } }`
var headers = map[string][]string{
	//	"x-rh-identity": {base64.StdEncoding.EncodeToString([]byte(identity))},
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	flag.Var(&headerFlags, "header", "header name:value")
	connectionCount := flag.Int("connection_count", 1, "number of connections to create")
	flag.Parse()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u, err := url.Parse(*targetUrl)
	if err != nil {
		log.Fatal("unable to parse url:", err)
	}
	log.Printf("connecting to %s\n", u.String())

	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < *connectionCount; i++ {
		go func(i int) {

			//backendSession, err := backends.NewTCPDialer(u.String(), false, nil)
			backendSession, err := backends.NewWebsocketDialer(u.String(), nil, headerFlags[0], false)
			if err != nil {
				log.Fatal("New Dialer error:", err)
				return
			}

			nodeID := fmt.Sprintf("node-%d", i)

			netceptorObj := netceptor.New(ctx, nodeID, nil)

			err = netceptorObj.AddBackend(backendSession, 1, nil)
			if err != nil {
				log.Fatal("AddBackend error:", err)
				return
			}

			log.Println("connected ", i)
			netceptorObj.BackendWait()
			log.Println("shutting down ", i)
		}(i)
	}

	<-c
	cancel()
}
