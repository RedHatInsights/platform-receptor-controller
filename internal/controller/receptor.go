package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/queue"
	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"
	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

type ReceptorService struct {
	AccountNumber string
	NodeID        string
	PeerNodeID    string

	Metadata interface{}

	// FIXME:  Move the channels into a Transport object/struct
	SendChannel    chan<- Message
	ControlChannel chan<- protocol.Message
	ErrorChannel   chan<- error

	cbrd *ChannelBasedResponseDispatcher
	/*
	   edges
	   seen
	*/
}

func (r *ReceptorService) RegisterConnection(peerNodeID string, metadata interface{}) error {
	log.Printf("Registering a connection to node %s", peerNodeID)

	r.PeerNodeID = peerNodeID
	r.Metadata = metadata

	r.cbrd = &ChannelBasedResponseDispatcher{
		DispatchTable: make(map[uuid.UUID]chan ResponseMessage),
	}

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

	r.SendChannel <- msg

	return &jobID, nil
}

func (r *ReceptorService) SendMessageSync(ctx context.Context, recipient string, route []string, payload interface{}, directive string) (interface{}, error) {

	jobID, err := r.SendMessage(recipient, route, payload, directive)
	if err != nil {
		log.Println("Unable to generate UUID for routing the job...cannot proceed")
		return nil, err
	}

	responseChannel := make(chan ResponseMessage)

	log.Println("Registering a sync response handler")
	r.cbrd.Register(*jobID, responseChannel)

	log.Println("Waiting for a sync response")
	select {
	case <-ctx.Done(): // i think the context could be setup with a timeout on the other end...

		r.cbrd.Unregister(*jobID)

		return nil, errors.New("RECPTOR SHUTDOWN")

	case responseMsg := <-responseChannel:

		r.cbrd.Unregister(*jobID)

		return responseMsg, nil

	case <-time.After(time.Second * 3):

		r.cbrd.Unregister(*jobID)

		log.Printf("**** waited too long on a message!!")
		return nil, errors.New("TIMEOUT!!")
	}

	return nil, nil
}

func (r *ReceptorService) DispatchResponse(payloadMessage *protocol.PayloadMessage) {

	responseMessage := ResponseMessage{
		AccountNumber: r.AccountNumber,
		Sender:        payloadMessage.RoutingInfo.Sender,
		MessageID:     payloadMessage.Data.MessageID,
		MessageType:   payloadMessage.Data.MessageType,
		Payload:       payloadMessage.Data.RawPayload,
		Code:          payloadMessage.Data.Code,
		InResponseTo:  payloadMessage.Data.InResponseTo,
		Serial:        payloadMessage.Data.Serial,
	}

	inResponseTo, _ := uuid.Parse(payloadMessage.Data.InResponseTo)

	responseChannel, _ := r.cbrd.GetDispatchChannel(inResponseTo)

	if responseChannel != nil {
		responseChannel <- responseMessage
		return
	}

	log.Printf("Dispatching response:%+v", responseMessage)

	jsonResponseMessage, err := json.Marshal(responseMessage)
	if err != nil {
		log.Println("JSON marshal of ResponseMessage failed, err:", err)
		return
	}

	// FIXME:
	ctx := context.Background()

	kw := queue.StartProducer(queue.GetProducer())
	kw.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(payloadMessage.Data.InResponseTo),
			Value: jsonResponseMessage,
		})

}

func (r *ReceptorService) Close() {
}

func (r *ReceptorService) DisconnectReceptorNetwork() {
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

type ChannelBasedResponseDispatcher struct {
	DispatchTable map[uuid.UUID]chan ResponseMessage
	sync.Mutex
}

func (cbrd *ChannelBasedResponseDispatcher) Register(msgID uuid.UUID, responseChannel chan ResponseMessage) {
	cbrd.Lock()
	cbrd.DispatchTable[msgID] = responseChannel
	cbrd.Unlock()
}

func (cbrd *ChannelBasedResponseDispatcher) Unregister(msgID uuid.UUID) {
	cbrd.Lock()
	delete(cbrd.DispatchTable, msgID)
	cbrd.Unlock()
}

func (cbrd *ChannelBasedResponseDispatcher) GetDispatchChannel(msgID uuid.UUID) (chan ResponseMessage, error) {
	var dispatchChannel chan ResponseMessage

	cbrd.Lock()
	dispatchChannel, _ = cbrd.DispatchTable[msgID]
	cbrd.Unlock()

	return dispatchChannel, nil
}
