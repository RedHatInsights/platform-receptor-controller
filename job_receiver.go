package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"net/http"
)

type JobReceiver struct {
	connectionMgr *ConnectionManager
	router        *http.ServeMux
}

func newJobReceiver(cm *ConnectionManager, r *http.ServeMux) *JobReceiver {
	return &JobReceiver{
		connectionMgr: cm,
		router:        r,
	}
}

func (jr *JobReceiver) routes() {
	jr.router.HandleFunc("/job", jr.handleJob())
}

func (jr *JobReceiver) handleJob() http.HandlerFunc {

	type JobRequest struct {
		Account   string `json:"account"`
		Recipient string `json:"recipient"`
		Payload   string `json:"payload"`
		Directive string `json:"directive"`
	}

	type JobResponse struct {
		JobID string `json:"id"`
	}

	return func(w http.ResponseWriter, req *http.Request) {

		fmt.Println("Simulating JobReceiver producing a message")

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

		// dispatch job
		var client Client
		client = jr.connectionMgr.GetConnection(jobRequest.Account)
		if client == nil {
			// FIXME: the connection to the client was not available
			fmt.Println("No connection to the customer...")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		jobID, err := uuid.NewUUID()
		if err != nil {
			fmt.Println("Unable to generate UUID for routing the job...cannot proceed")
			return
		}

		jobResponse := JobResponse{jobID.String()}

		fmt.Println("job request:", jobRequest)

		client.SendWork([]byte("blah..."))

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(jobResponse); err != nil {
			panic(err)
		}
	}
}
