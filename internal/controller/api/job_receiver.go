package api

import (
	"github.com/gorilla/mux"

	//	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/redhatinsights/platform-go-middlewares/identity"

	kafka "github.com/segmentio/kafka-go"
)

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
				panic(err)
			}
		}

		jobResponse := JobResponse{jobID.String()}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(jobResponse); err != nil {
			panic(err)
		}
	}
}
