package api

import (
	"fmt"
	"io"

	"github.com/gorilla/mux"

	//	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/go-playground/validator/v10"
	"github.com/redhatinsights/platform-go-middlewares/identity"

	kafka "github.com/segmentio/kafka-go"
)

type badRequest struct {
	status int
	msg    string
}

func (br *badRequest) Error() string {
	return fmt.Sprintf("%d: %s", br.status, br.msg)
}

func decodeJSON(body io.ReadCloser, job interface{}) error {
	dec := json.NewDecoder(body)
	if err := dec.Decode(&job); err != nil {
		// FIXME: More specific error handling needed.. case statement for different scenarios?
		return &badRequest{status: http.StatusBadRequest, msg: "Request body includes malformed json"}
	}

	v := validator.New()
	if err := v.Struct(job); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			log.Println(e)
		}
		return &badRequest{status: http.StatusBadRequest, msg: "Request body is missing required fields"}
	} else if dec.More() {
		return &badRequest{status: http.StatusBadRequest, msg: "Request body must only contain one json object"}
	}

	return nil
}

type JobReceiver struct {
	connectionMgr *controller.ConnectionManager
	router        *mux.Router
	producer      *kafka.Writer
}

func NewJobReceiver(cm *controller.ConnectionManager, r *mux.Router, kw *kafka.Writer) *JobReceiver {
	return &JobReceiver{
		connectionMgr: cm,
		router:        r,
		producer:      kw,
	}
}

func (jr *JobReceiver) Routes() {
	securedSubRouter := jr.router.PathPrefix("/").Subrouter()
	securedSubRouter.Use(identity.EnforceIdentity)
	securedSubRouter.HandleFunc("/job", jr.handleJob())
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
				return
			}
			return
		}

		log.Println("jobRequest:", jobRequest)
		// dispatch job via client's sendwork
		// not using client's sendwork, but leaving this code in to verify connection?
		var client controller.Client
		client = jr.connectionMgr.GetConnection(jobRequest.Account, jobRequest.Recipient)
		if client == nil {
			// FIXME: the connection to the client was not available
			log.Println("No connection to the customer...")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		jobID, err := uuid.NewRandom()
		if err != nil {
			log.Println("Unable to generate UUID for routing the job...cannot proceed")
			return
		}

		jobResponse := JobResponse{jobID.String()}

		log.Println("job request:", jobRequest)

		workRequest := controller.Message{MessageID: jobID,
			Recipient: jobRequest.Recipient,
			RouteList: []string{jobRequest.Recipient},
			Payload:   jobRequest.Payload,
			Directive: jobRequest.Directive}

		client.SendMessage(workRequest)

		/*
			// dispatch job via kafka queue
			jobRequestJSON, err := json.Marshal(jobRequest)
			jr.producer.WriteMessages(context.Background(),
				kafka.Message{
					Key:   []byte(jobRequest.Account),
					Value: []byte(jobRequestJSON),
				})
		*/

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(jobResponse); err != nil {
			panic(err)
		}
	}
}
