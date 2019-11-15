package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type ManagementServer struct {
	connection_mgr *ConnectionManager
	router         *http.ServeMux
}

func newManagementServer(cm *ConnectionManager, r *http.ServeMux) *ManagementServer {
	return &ManagementServer{
		connection_mgr: cm,
		router:         r,
	}
}

func (s *ManagementServer) routes() {
	s.router.HandleFunc("/management/disconnect", s.handleDisconnect())
}

func (s *ManagementServer) handleDisconnect() http.HandlerFunc {

	type connectionId struct {
		Account string `json: "account"`
		Node_id string `json: "node_id"`
	}

	return func(w http.ResponseWriter, req *http.Request) {

		var conn_id connectionId

		body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))

		if err != nil {
			panic(err)
		}

		if err := req.Body.Close(); err != nil {
			panic(err)
		}

		if err := json.Unmarshal(body, &conn_id); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(422) // unprocessable entity
			if err := json.NewEncoder(w).Encode(err); err != nil {
				panic(err)
			}
		}

		fmt.Println(conn_id)

		client := s.connection_mgr.GetConnection(conn_id.Account)
		if client == nil {
			w.WriteHeader(404)
			fmt.Printf("No connection to the customer (%+v)...\n", conn_id)
			return
		}

		client.DisconnectReceptorNetwork()

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(conn_id); err != nil {
			panic(err)
		}
	}
}
