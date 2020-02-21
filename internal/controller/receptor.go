package controller

type Receptor struct {
}

type ReceptorStateMachine struct {
	handshakeComplete    bool
	routingTableReceived bool
}
