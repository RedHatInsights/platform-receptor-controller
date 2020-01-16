package job_router

import "log"

// FIXME: Decide on concurrency approach here

type JobRouter struct {
	// FIXME: this only allows for one connection per account
	connection_channels map[string]chan []byte
}

func NewJobRouter() *JobRouter {
	return &JobRouter{
		connection_channels: make(map[string]chan []byte),
	}
}

func (jr *JobRouter) DispatchJob(account string, job []byte /*FIXME: what is the correct "job" type*/) {
	// FIXME: this only allows for one connection per account

	log.Println("Routing job to account:", account)

	job_channel, exists := jr.connection_channels[account]
	if exists == false {
		// FIXME:
		log.Println("ERROR: connection to customer does not exist")
		return
	}

	// FIXME: add some validation checks to make sure we are sending the jobs to correct customer

	go func() {
		log.Println("Go routine ... sending job to connection...")
		job_channel <- job
	}()
}

func (jr *JobRouter) RegisterReceptorNetwork(account string, c chan []byte) {
	log.Println("Registering a receptor-collector network")
	// FIXME: this only allows for one connection per account
	jr.connection_channels[account] = c
}

func (jr *JobRouter) UnregisterReceptorNetwork(account string) {
	log.Println("Unregistering a receptor-collector network")

	// FIXME: this only allows for one connection per account

	// FIXME: What about multiple connections per account??
	delete(jr.connection_channels, account)
}
