package api

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/middlewares"

	"github.com/gorilla/mux"
)

const (
	CONNECTED_STATUS    = "connected"
	DISCONNECTED_STATUS = "disconnected"
)

type ManagementServer struct {
	connectionMgr *controller.ConnectionManager
	router        *mux.Router
	secrets       map[string]interface{}
}

func NewManagementServer(cm *controller.ConnectionManager, r *mux.Router, secrets map[string]interface{}) *ManagementServer {
	return &ManagementServer{
		connectionMgr: cm,
		router:        r,
		secrets:       secrets,
	}
}

func (s *ManagementServer) Routes() {
	securedSubRouter := s.router.PathPrefix("/connection").Subrouter()
	amw := &middlewares.AuthMiddleware{Secrets: s.secrets}
	securedSubRouter.Use(amw.Authenticate)
	securedSubRouter.HandleFunc("", s.handleConnectionListing()).Methods(http.MethodGet)
	securedSubRouter.HandleFunc("/disconnect", s.handleDisconnect()).Methods(http.MethodPost)
	securedSubRouter.HandleFunc("/status", s.handleConnectionStatus()).Methods(http.MethodPost)
	securedSubRouter.HandleFunc("/ping", s.handleConnectionPing()).Methods(http.MethodPost)
}

type errorResponse struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

type connectionID struct {
	Account string `json:"account"`
	NodeID  string `json:"node_id"`
}

type connectionStatusResponse struct {
	Status       string      `json:"status"`
	Capabilities interface{} `json:"capabilities,omitempty"`
}

type connectionPingResponse struct {
	Status  string      `json:"status"`
	Payload interface{} `json:"payload"`
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
			errorResponse := errorResponse{Title: "Unable to process json input",
				Status: http.StatusUnprocessableEntity,
				Detail: err.Error()}
			WriteJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client == nil {
			log.Printf("No connection to the customer (%+v)...\n", connID)
			errorResponse := errorResponse{Title: "No connection found to node",
				Status: http.StatusBadRequest,
				Detail: "No connection found to node"}
			WriteJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		client.Close()

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
			errorResponse := errorResponse{Title: "Unable to process json input",
				Status: http.StatusUnprocessableEntity,
				Detail: err.Error()}
			WriteJSONResponse(w, errorResponse.Status, errorResponse)
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
			errorResponse := errorResponse{Title: "Unable to process json input",
				Status: http.StatusUnprocessableEntity,
				Detail: err.Error()}
			WriteJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		log.Println(connID)

		pingResponse := connectionPingResponse{Status: DISCONNECTED_STATUS}
		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client == nil {
			WriteJSONResponse(w, http.StatusOK, pingResponse)
			return
		}

		pingResponse.Status = CONNECTED_STATUS
		pingResponse.Payload, err = client.Ping(req.Context(), connID.NodeID, []string{connID.NodeID})
		if err != nil {
			errorResponse := errorResponse{Title: "Ping failed",
				Status: http.StatusBadRequest,
				Detail: err.Error()}
			WriteJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		WriteJSONResponse(w, http.StatusOK, pingResponse)
	}
}

func (s *ManagementServer) handleConnectionListing() http.HandlerFunc {

	type ConnectionsPerAccount struct {
		AccountNumber string   `json:"account"`
		Connections   []string `json:"connections"`
	}

	type Response struct {
		Connections []ConnectionsPerAccount `json:"connections"`
	}

	return func(w http.ResponseWriter, req *http.Request) {

		allReceptorConnections := s.connectionMgr.GetAllConnections()
		log.Println("allReceptorConnections:", allReceptorConnections)

		connections := make([]ConnectionsPerAccount, len(allReceptorConnections))

		accountCount := 0
		for key, value := range allReceptorConnections {
			connections[accountCount].AccountNumber = key
			connections[accountCount].Connections = make([]string, len(value))
			nodeCount := 0
			for k, _ := range value {
				connections[accountCount].Connections[nodeCount] = k
				nodeCount++
			}

			accountCount++
		}

		response := Response{Connections: connections}

		log.Println("response:", response)

		WriteJSONResponse(w, http.StatusOK, response)
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
