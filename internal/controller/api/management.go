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
	securedSubRouter.HandleFunc("/ping", s.handleConnectionPing())
}

type connectionID struct {
	Account string `json:"account"`
	NodeID  string `json:"node_id"`
}

type connectionStatusResponse struct {
	Status       string      `json:"status"`
	Capabilities interface{} `json:"capabilities,omitempty"`
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
			WriteJSONResponse(w, http.StatusUnprocessableEntity, err)
		}

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client == nil {
			connectionStatus := connectionStatusResponse{Status: DISCONNECTED_STATUS}
			WriteJSONResponse(w, http.StatusNotFound, connectionStatus)
			log.Printf("No connection to the customer (%+v)...\n", connID)
			return
		}

		client.DisconnectReceptorNetwork()

		WriteJSONResponse(w, http.StatusOK, struct{}{})
	}
}

func (s *ManagementServer) handleConnectionStatus() http.HandlerFunc {

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
			WriteJSONResponse(w, http.StatusUnprocessableEntity, err)
			return
		}

		log.Println(connID)

		var connectionStatus connectionStatusResponse

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client != nil {
			connectionStatus.Status = CONNECTED_STATUS
			connectionStatus.Capabilities = client.GetCapabilities()
		} else {
			connectionStatus.Status = DISCONNECTED_STATUS
		}

		WriteJSONResponse(w, http.StatusOK, connectionStatus)
	}
}

func (s *ManagementServer) handleConnectionPing() http.HandlerFunc {

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
			WriteJSONResponse(w, http.StatusUnprocessableEntity, err)
			return
		}

		log.Println(connID)

		var payload interface{}

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client == nil {
			connectionStatus := connectionStatusResponse{Status: DISCONNECTED_STATUS}
			WriteJSONResponse(w, http.StatusOK, connectionStatus)
			return
		}

		payload, err = client.Ping(req.Context(), connID.NodeID, []string{connID.NodeID})
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestTimeout)
			return
		}

		WriteJSONResponse(w, http.StatusOK, payload)
	}
}

func WriteJSONResponse(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Unable to encode payload!", http.StatusUnprocessableEntity)
		log.Println("Unable to encode payload!")
		return
	}
}
