package api

import (
	"github.com/gorilla/mux"

	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/middlewares"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	kafka "github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type JobReceiver struct {
	connectionMgr *controller.ConnectionManager
	router        *mux.Router
	producer      *kafka.Writer
	config        *config.Config
}

func NewJobReceiver(cm *controller.ConnectionManager, r *mux.Router, kw *kafka.Writer, cfg *config.Config) *JobReceiver {
	return &JobReceiver{
		connectionMgr: cm,
		router:        r,
		producer:      kw,
		config:        cfg,
	}
}

func (jr *JobReceiver) Routes() {
	securedSubRouter := jr.router.PathPrefix("/").Subrouter()
	amw := &middlewares.AuthMiddleware{Secrets: jr.config.ServiceToServiceCredentials}
	securedSubRouter.Use(amw.Authenticate)
	securedSubRouter.HandleFunc("/job", jr.handleJob()).Methods(http.MethodPost)
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
		logger := logger.Log.WithFields(logrus.Fields{
			"account":    principal.GetAccount(),
			"request_id": requestId})

		var jobRequest JobRequest

		body := http.MaxBytesReader(w, req.Body, 1048576)

		if err := decodeJSON(body, &jobRequest); err != nil {
			errMsg := "Unable to process json input"
			logger.WithFields(logrus.Fields{"error": err}).Debug(errMsg)
			errorResponse := errorResponse{Title: errMsg,
				Status: http.StatusBadRequest,
				Detail: err.Error()}
			writeJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		var client controller.Receptor
		client = jr.connectionMgr.GetConnection(jobRequest.Account, jobRequest.Recipient)
		if client == nil {
			// The connection to the customer's receptor node was not available
			errMsg := "No connection to the receptor node"
			logger.Info(errMsg)
			errorResponse := errorResponse{Title: errMsg,
				Status: http.StatusNotFound,
				Detail: errMsg}
			writeJSONResponse(w, errorResponse.Status, errorResponse)
			return
		}

		logger = logger.WithFields(logrus.Fields{"recipient": jobRequest.Recipient,
			"directive": jobRequest.Directive})
		logger.Debug("Sending a message")

		jobID, err := client.SendMessage(req.Context(), jobRequest.Recipient,
			[]string{jobRequest.Recipient},
			jobRequest.Payload,
			jobRequest.Directive)

		if err != nil {
			// FIXME:  Handle this better!?!?
			logger.WithFields(logrus.Fields{"error": err}).Info("Error passing message to receptor")
			errorResponse := errorResponse{Title: "Error passing message to receptor",
				Status: http.StatusUnprocessableEntity,
				Detail: err.Error()}
			writeJSONResponse(w, errorResponse.Status, errorResponse)
		}

		logger.WithFields(logrus.Fields{"message_id": jobID}).Debug("Message sent")

		jobResponse := JobResponse{jobID.String()}

		writeJSONResponse(w, http.StatusCreated, jobResponse)
	}
}
