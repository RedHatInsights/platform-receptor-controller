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
	s.router.HandleFunc("/management/disconnect", s.handleManagement())
}

func (s *ManagementServer) handleManagement() http.HandlerFunc {

	// FIXME: Rename me
	type Job struct {
		Account string `json: "account"`
		Node_id string `json: "node_id"`
	}

	return func(w http.ResponseWriter, req *http.Request) {

		fmt.Println("Simulating ManagementServer producing a message")

		var job Job

		body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))

		if err != nil {
			panic(err)
		}

		if err := req.Body.Close(); err != nil {
			panic(err)
		}

		if err := json.Unmarshal(body, &job); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(422) // unprocessable entity
			if err := json.NewEncoder(w).Encode(err); err != nil {
				panic(err)
			}
		}

        fmt.Println(job)

		// dispatch job
		var client Client
		client = s.connection_mgr.GetConnection(job.Account)
		if client == nil {
			// FIXME:  the connection is not connected!!
			//         is it connected to another pod?
			fmt.Println("Not sure what to do here!!")
			fmt.Println("No connection to the customer...leaving")
			return
		}

		// FIXME: just for testing
		client.DisconnectReceptorNetwork()

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(job); err != nil {
			panic(err)
		}
	}
}
