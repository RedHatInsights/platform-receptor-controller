package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type ManagementServer struct {
	connectionMgr *ConnectionManager
	router        *http.ServeMux
}

func newManagementServer(cm *ConnectionManager, r *http.ServeMux) *ManagementServer {
	return &ManagementServer{
		connectionMgr: cm,
		router:        r,
	}
}

func (s *ManagementServer) routes() {
	s.router.HandleFunc("/management/disconnect", s.handleDisconnect())
}

func (s *ManagementServer) handleDisconnect() http.HandlerFunc {

	type connectionID struct {
		Account string `json: "account"`
		NodeID  string `json: "node_id"`
	}

	return func(w http.ResponseWriter, req *http.Request) {

		var connID connectionID

		body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))

		if err != nil {
			panic(err)
		}

		if err := req.Body.Close(); err != nil {
			panic(err)
		}

		if err := json.Unmarshal(body, &connID); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(422) // unprocessable entity
			if err := json.NewEncoder(w).Encode(err); err != nil {
				panic(err)
			}
		}

		fmt.Println(connID)

		client := s.connectionMgr.GetConnection(connID.Account)
		if client == nil {
			w.WriteHeader(404)
			fmt.Printf("No connection to the customer (%+v)...\n", connID)
			return
		}

		client.DisconnectReceptorNetwork()

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(connID); err != nil {
			panic(err)
		}
	}
}
