package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	kafka "github.com/segmentio/kafka-go"
)

type Job struct {
	Account   string `json:"account"`
	MessageID string `json:"message_id"`
	Recipient string `json:"recipient"`
	Payload   string `json:"payload"`
	Directive string `json:"directive"`
}

type JobReceiver struct {
	connectionMgr *ConnectionManager
	router        *http.ServeMux
	producer      *kafka.Writer
}

func newJobReceiver(cm *ConnectionManager, r *http.ServeMux, kw *kafka.Writer) *JobReceiver {
	return &JobReceiver{
		connectionMgr: cm,
		router:        r,
		producer:      kw,
	}
}

func (jr *JobReceiver) routes() {
	jr.router.HandleFunc("/job", jr.handleJob())
}

func (jr *JobReceiver) handleJob() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		fmt.Println("Simulating JobReceiver producing a message")

		var job Job

		body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1048576))

		if err != nil {
			panic(err)
		}

		if err := req.Body.Close(); err != nil {
			panic(err)
		}

		if err := json.Unmarshal(body, &job); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(422) // unprocessable entity
			if err := json.NewEncoder(w).Encode(err); err != nil {
				panic(err)
			}
		}

		// dispatch job via client's sendwork
		// not using client's sendwork, but leaving this code in to verify connection?
		var client Client
		client = jr.connectionMgr.GetConnection(job.Account)
		if client == nil {
			// FIXME:  the connection is not connected!!
			//         is it connected to another pod?
			fmt.Println("Not sure what to do here!!")
			fmt.Println("No connection to the customer...leaving")
			return
		}
		// client.SendWork([]byte("blah..."))

		// dispatch job via kafka queue
		jobJSON, err := json.Marshal(job)
		jr.producer.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(job.Account),
				Value: []byte(jobJSON),
			})

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(job); err != nil {
			panic(err)
		}
	}
}
