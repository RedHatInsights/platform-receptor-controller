package controller

import (
	"log"
)

type Receptor struct {
	Account    string
	NodeID     string
	PeerNodeID string

	Metadata interface{}

	/*
	   edges
	   seen
	*/
}

func (r *Receptor) RegisterConnection(peerNodeID string, metadata interface{}) error {
	log.Printf("Registering a connection to node %s", peerNodeID)

	r.PeerNodeID = peerNodeID
	r.Metadata = metadata

	return nil
}

func (r *Receptor) UpdateRoutingTable(edges string, seen string) error {
	log.Println("edges:", edges)
	log.Println("seen:", seen)

	return nil
}
