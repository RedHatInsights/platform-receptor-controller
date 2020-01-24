package controller

import (
	"github.com/gorilla/mux"
	"github.com/redhatinsights/platform-go-middlewares/identity"

	//	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/uuid"

	kafka "github.com/segmentio/kafka-go"
)

type JobReceiver struct {
	connectionMgr *ConnectionManager
	router        *mux.Router
	producer      *kafka.Writer
}

func NewJobReceiver(cm *ConnectionManager, r *mux.Router, kw *kafka.Writer) *JobReceiver {
	return &JobReceiver{
		connectionMgr: cm,
		router:        r,
		producer:      kw,
	}
}

func (jr *JobReceiver) Routes() {
	jr.router.HandleFunc("/job", jr.handleJob())
	jr.router.Use(identity.EnforceIdentity)
}

func (jr *JobReceiver) handleJob() http.HandlerFunc {

	type JobRequest struct {
		Account   string      `json:"account"`
		Recipient string      `json:"recipient"`
		Payload   interface{} `json:"payload"`
		Directive string      `json:"directive"`
	}

	type JobResponse struct {
		JobID string `json:"id"`
	}

	return func(w http.ResponseWriter, req *http.Request) {

		log.Println("Simulating JobReceiver producing a message")

		var jobRequest JobRequest

		body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))

		if err != nil {
			panic(err)
		}

		if err := req.Body.Close(); err != nil {
			panic(err)
		}

		if err := json.Unmarshal(body, &jobRequest); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err := json.NewEncoder(w).Encode(err); err != nil {
				panic(err)
			}
		}

		log.Println("jobRequest:", jobRequest)
		// dispatch job via client's sendwork
		// not using client's sendwork, but leaving this code in to verify connection?
		var client Client
		client = jr.connectionMgr.GetConnection(jobRequest.Account, jobRequest.Recipient)
		if client == nil {
			// FIXME: the connection to the client was not available
			log.Println("No connection to the customer...")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		jobID, err := uuid.NewUUID()
		if err != nil {
			log.Println("Unable to generate UUID for routing the job...cannot proceed")
			return
		}

		jobResponse := JobResponse{jobID.String()}

		log.Println("job request:", jobRequest)

		workRequest := Work{MessageID: jobID.String(),
			Recipient: jobRequest.Recipient,
			RouteList: []string{"node-b", "node-a"},
			Payload:   jobRequest.Payload,
			Directive: jobRequest.Directive}

		client.SendWork(workRequest)

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
