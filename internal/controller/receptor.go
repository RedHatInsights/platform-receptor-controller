package controller

type MeshConnection struct {
	peerNodeID string
}

type ReceptorStateMachine struct {
	handshakeComplete    bool
	routingTableReceived bool

	account    string
	peerNodeID string

	connections map[string]MeshConnection

	capabilities interface{}
	/*
	   edges
	   seen
	*/
}
