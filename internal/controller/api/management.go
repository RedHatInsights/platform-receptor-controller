package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/redhatinsights/platform-go-middlewares/identity"

	"github.com/gorilla/mux"
)

const (
	CONNECTED_STATUS    = "connected"
	DISCONNECTED_STATUS = "disconnected"
)

type ManagementServer struct {
	connectionMgr *controller.ConnectionManager
	router        *mux.Router
}

func NewManagementServer(cm *controller.ConnectionManager, r *mux.Router) *ManagementServer {
	return &ManagementServer{
		connectionMgr: cm,
		router:        r,
	}
}

func (s *ManagementServer) Routes() {
	securedSubRouter := s.router.PathPrefix("/connection").Subrouter()
	securedSubRouter.Use(identity.EnforceIdentity)
	securedSubRouter.HandleFunc("/disconnect", s.handleDisconnect())
	securedSubRouter.HandleFunc("/status", s.handleConnectionStatus())
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

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client == nil {
			w.WriteHeader(http.StatusNotFound)
			log.Printf("No connection to the customer (%+v)...\n", connID)
			return
		}

		client.DisconnectReceptorNetwork()

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
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
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err := json.NewEncoder(w).Encode(err); err != nil {
				panic(err)
			}
		}

		log.Println(connID)

		var connectionStatus connectionStatusResponse

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client != nil {
			connectionStatus.Status = CONNECTED_STATUS
		} else {
			connectionStatus.Status = DISCONNECTED_STATUS
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(connectionStatus); err != nil {
			panic(err)
		}
	}
}
