package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/middlewares"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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

		principal, _ := middlewares.GetPrincipal(req.Context())
		requestId := request_id.GetReqID(req.Context())
		logger := logger.Log.WithFields(logrus.Fields{
			"account":    principal.GetAccount(),
			"request_id": requestId})

		body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))

		if err != nil {
			panic(err)
		}

		if err := req.Body.Close(); err != nil {
			panic(err)
		}

		var connID connectionID

		if err := json.Unmarshal(body, &connID); err != nil {
			errorResponse := errorResponse{Title: "Unable to process json input",
				Status: http.StatusUnprocessableEntity,
				Detail: err.Error()}
			writeJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client == nil {
			errMsg := fmt.Sprintf("No connection found for node (%s:%s)", connID.Account, connID.NodeID)
			logger.Info(errMsg)
			errorResponse := errorResponse{Title: errMsg,
				Status: http.StatusBadRequest,
				Detail: errMsg}
			writeJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		logger.Infof("Attempting to disconnect account:%s - node id:%s",
			connID.Account, connID.NodeID)

		client.Close()

		writeJSONResponse(w, http.StatusOK, struct{}{})
	}
}

func (s *ManagementServer) handleConnectionStatus() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		principal, _ := middlewares.GetPrincipal(req.Context())
		requestId := request_id.GetReqID(req.Context())
		logger := logger.Log.WithFields(logrus.Fields{
			"account":    principal.GetAccount(),
			"request_id": requestId})

		body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))

		if err != nil {
			panic(err)
		}

		if err := req.Body.Close(); err != nil {
			panic(err)
		}

		var connID connectionID
		if err := json.Unmarshal(body, &connID); err != nil {
			errorResponse := errorResponse{Title: "Unable to process json input",
				Status: http.StatusUnprocessableEntity,
				Detail: err.Error()}
			writeJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		logger.Debugf("Checking connection status for account:%s - node id:%s",
			connID.Account, connID.NodeID)

		var connectionStatus connectionStatusResponse

		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client != nil {
			connectionStatus.Status = CONNECTED_STATUS
			connectionStatus.Capabilities = client.GetCapabilities()
		} else {
			connectionStatus.Status = DISCONNECTED_STATUS
		}

		writeJSONResponse(w, http.StatusOK, connectionStatus)
	}
}

func (s *ManagementServer) handleConnectionPing() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		principal, _ := middlewares.GetPrincipal(req.Context())
		requestId := request_id.GetReqID(req.Context())
		logger := logger.Log.WithFields(logrus.Fields{
			"account":    principal.GetAccount(),
			"request_id": requestId})

		body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))

		if err != nil {
			panic(err)
		}

		if err := req.Body.Close(); err != nil {
			panic(err)
		}

		var connID connectionID
		if err := json.Unmarshal(body, &connID); err != nil {
			errorResponse := errorResponse{Title: "Unable to process json input",
				Status: http.StatusUnprocessableEntity,
				Detail: err.Error()}
			writeJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		logger.Debugf("Submitting ping for account:%s - node id:%s",
			connID.Account, connID.NodeID)

		pingResponse := connectionPingResponse{Status: DISCONNECTED_STATUS}
		client := s.connectionMgr.GetConnection(connID.Account, connID.NodeID)
		if client == nil {
			writeJSONResponse(w, http.StatusOK, pingResponse)
			return
		}

		pingResponse.Status = CONNECTED_STATUS
		pingResponse.Payload, err = client.Ping(req.Context(), connID.NodeID, []string{connID.NodeID})
		if err != nil {
			errorResponse := errorResponse{Title: "Ping failed",
				Status: http.StatusBadRequest,
				Detail: err.Error()}
			writeJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		writeJSONResponse(w, http.StatusOK, pingResponse)
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

		principal, _ := middlewares.GetPrincipal(req.Context())
		requestId := request_id.GetReqID(req.Context())
		logger := logger.Log.WithFields(logrus.Fields{
			"account":    principal.GetAccount(),
			"request_id": requestId})

		logger.Debugf("Getting connection list")

		allReceptorConnections := s.connectionMgr.GetAllConnections()

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

		writeJSONResponse(w, http.StatusOK, response)
	}
}
