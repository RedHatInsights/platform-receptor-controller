package api

import (
	"errors"
	"io"

	"github.com/gorilla/mux"

	//	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/middlewares"
	"github.com/go-playground/validator/v10"

	kafka "github.com/segmentio/kafka-go"
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

		log.Println("Simulating JobReceiver producing a message")

		var jobRequest JobRequest

		body := http.MaxBytesReader(w, req.Body, 1048576)

		if err := decodeJSON(body, &jobRequest); err != nil {
			log.Println(err)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(err); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			return
		}

		log.Println("jobRequest:", jobRequest)
		// dispatch job via client's sendwork
		// not using client's sendwork, but leaving this code in to verify connection?
		var client controller.Receptor
		client = jr.connectionMgr.GetConnection(jobRequest.Account, jobRequest.Recipient)
		if client == nil {
			// FIXME: the connection to the client was not available
			log.Println("No connection to the customer...")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		log.Println("job request:", jobRequest)

		jobID, err := client.SendMessage(req.Context(), jobRequest.Recipient,
			[]string{jobRequest.Recipient},
			jobRequest.Payload,
			jobRequest.Directive)

		if err != nil {
			// FIXME:  Handle this better!?!?
			log.Println("Error passing message to receptor: ", err)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err := json.NewEncoder(w).Encode(err); err != nil {
				http.Error(w, err.Error(), 500)
				return

			}
		}

		jobResponse := JobResponse{jobID.String()}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(jobResponse); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
}
