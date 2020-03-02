package controller

import (
	"log"

	"github.com/google/uuid"
)

type ReceptorService struct {
	AccountNumber string
	NodeID        string
	PeerNodeID    string

	Metadata interface{}

	Transport *Transport

	/*
	   edges
	   seen
	*/
}

func (r *ReceptorService) RegisterConnection(peerNodeID string, metadata interface{}) error {
	log.Printf("Registering a connection to node %s", peerNodeID)

	r.PeerNodeID = peerNodeID
	r.Metadata = metadata

	return nil
}

func (r *ReceptorService) UpdateRoutingTable(edges string, seen string) error {
	log.Println("edges:", edges)
	log.Println("seen:", seen)

	return nil
}

func (r *ReceptorService) SendMessage(recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {

	jobID, err := uuid.NewRandom()
	if err != nil {
		log.Println("Unable to generate UUID for routing the job...cannot proceed")
		return nil, err
	}

	msg := Message{MessageID: jobID,
		Recipient: recipient,
		RouteList: route,
		Payload:   payload,
		Directive: directive}

	r.Transport.Send <- msg

	return &jobID, nil
}

func (r *ReceptorService) Close() {
	r.Transport.Cancel()
}

func (r *ReceptorService) DisconnectReceptorNetwork() {
	r.Transport.Cancel()
}

func (r *ReceptorService) GetCapabilities() interface{} {
	emptyCapabilities := struct{}{}

	if r.Metadata == nil {
		return emptyCapabilities
	}

	metadata, ok := r.Metadata.(map[string]interface{})
	if ok != true {
		return emptyCapabilities
	}

	capabilities, exist := metadata["capabilities"]
	if exist != true {
		return emptyCapabilities
	}

	return capabilities
}
