package api

import (
	"errors"
	"io"

	"github.com/gorilla/mux"

	"encoding/json"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/middlewares"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/go-playground/validator/v10"
	kafka "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func decodeJSON(body io.ReadCloser, job interface{}) error {
	dec := json.NewDecoder(body)
	if err := dec.Decode(&job); err != nil {
		// FIXME: More specific error handling needed.. case statement for different scenarios?
		return errors.New("Request body includes malformed json")
	}

	v := validator.New()
	if err := v.Struct(job); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			log.Println(e)
		}
		return errors.New("Request body is missing required fields")
	} else if dec.More() {
		return errors.New("Request body must only contain one json object")
	}

	return nil
}

type JobReceiver struct {
	connectionMgr *controller.ConnectionManager
	router        *mux.Router
	producer      *kafka.Writer
	secrets       map[string]interface{}
}

func NewJobReceiver(cm *controller.ConnectionManager, r *mux.Router, kw *kafka.Writer, secrets map[string]interface{}) *JobReceiver {
	return &JobReceiver{
		connectionMgr: cm,
		router:        r,
		producer:      kw,
		secrets:       secrets,
	}
}

func (jr *JobReceiver) Routes() {
	securedSubRouter := jr.router.PathPrefix("/").Subrouter()
	amw := &middlewares.AuthMiddleware{Secrets: jr.secrets}
	securedSubRouter.Use(amw.Authenticate)
	securedSubRouter.HandleFunc("/job", jr.handleJob()).Methods("POST")
}

func (jr *JobReceiver) handleJob() http.HandlerFunc {

	type JobRequest struct {
		Account   string      `json:"account" validate:"required"`
		Recipient string      `json:"recipient" validate:"required"`
		Payload   interface{} `json:"payload" validate:"required"`
		Directive string      `json:"directive" validate:"required"`
	}

	type JobResponse struct {
		JobID string `json:"id"`
	}

	return func(w http.ResponseWriter, req *http.Request) {

		principal, _ := middlewares.GetPrincipal(req.Context())
		requestId := request_id.GetReqID(req.Context())
		requestLogger := logger.Log.WithFields(logrus.Fields{
			"account":    principal.GetAccount(),
			"request_id": requestId})

		var jobRequest JobRequest

		body := http.MaxBytesReader(w, req.Body, 1048576)

		if err := decodeJSON(body, &jobRequest); err != nil {
			errMsg := "Unable to process json input"
			requestLogger.WithFields(logrus.Fields{"error": err}).Debug(errMsg)
			errorResponse := errorResponse{Title: errMsg,
				Status: http.StatusBadRequest,
				Detail: err.Error()}
			WriteJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		var client controller.Receptor
		client = jr.connectionMgr.GetConnection(jobRequest.Account, jobRequest.Recipient)
		if client == nil {
			// The connection to the customer's receptor node was not available
			errMsg := "No connection to the receptor node"
			requestLogger.Info(errMsg)
			errorResponse := errorResponse{Title: errMsg,
				Status: http.StatusNotFound,
				Detail: errMsg}
			WriteJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		requestLogger.WithFields(logrus.Fields{"node_id": jobRequest.Recipient,
			"directive": jobRequest.Directive}).Debug("Sending a message:", jobRequest)

		jobID, err := client.SendMessage(req.Context(), jobRequest.Recipient,
			[]string{jobRequest.Recipient},
			jobRequest.Payload,
			jobRequest.Directive)

		if err != nil {
			// FIXME:  Handle this better!?!?
			requestLogger.WithFields(logrus.Fields{"error": err}).Info("Error passing message to receptor")
			errorResponse := errorResponse{Title: "Error passing message to receptor",
				Status: http.StatusUnprocessableEntity,
				Detail: err.Error()}
			WriteJSONResponse(w, errorResponse.Status, errorResponse)
		}

		jobResponse := JobResponse{jobID.String()}

		WriteJSONResponse(w, http.StatusCreated, jobResponse)
	}
}
