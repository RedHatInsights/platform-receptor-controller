package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

type ManagementServer struct {
	connectionMgr *ConnectionManager
	router        *mux.Router
}

func NewManagementServer(cm *ConnectionManager, r *mux.Router) *ManagementServer {
	return &ManagementServer{
		connectionMgr: cm,
		router:        r,
	}
}

func (s *ManagementServer) Routes() {
	s.router.HandleFunc("/management/disconnect", s.handleDisconnect())
	// FIXME: This might not belong here
	s.router.HandleFunc("/management/connection_status", s.handleConnectionStatus())
	s.router.Use(identity.EnforceIdentity)
}

type connectionID struct {
	Account string `json:"account"`
	NodeID  string `json:"node_id"`
}

func (s *ManagementServer) handleDisconnect() http.HandlerFunc {

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

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
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

// FIXME: This might not belong here
func (s *ManagementServer) handleConnectionStatus() http.HandlerFunc {

	type connectionStatusResponse struct {
		Status string `json:"status"`
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

		var connectionStatus connectionStatusResponse

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client != nil {
			connectionStatus.Status = "connected"
		} else {
			connectionStatus.Status = "disconnected"
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(connectionStatus); err != nil {
			panic(err)
		}
	}
}
