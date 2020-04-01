package controller

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"

	"github.com/sirupsen/logrus"
)

var (
	connectionToReceptorNetworkLost = errors.New("Connection to receptor network lost")
	requestCancelledBySender        = errors.New("Unable to complete the request.  Request cancelled by message sender.")
	requestTimedOut                 = errors.New("Unable to complete the request.  Request timed out.")
)

type ReceptorServiceFactory struct {
	kafkaWriter *kafka.Writer
}

func NewReceptorServiceFactory(w *kafka.Writer) *ReceptorServiceFactory {
	return &ReceptorServiceFactory{
		kafkaWriter: w,
	}
}

func (fact *ReceptorServiceFactory) NewReceptorService(logger *logrus.Entry, account, nodeID string, transport *Transport) *ReceptorService {
	return &ReceptorService{
		AccountNumber: account,
		NodeID:        nodeID,
		Transport:     transport,
		responseDispatcherRegistrar: &DispatcherTable{
			dispatchTable: make(map[uuid.UUID]chan ResponseMessage),
		},
		kafkaWriter: fact.kafkaWriter,
		logger:      logger,
	}
}

type ReceptorService struct {
	AccountNumber string
	NodeID        string
	PeerNodeID    string

	Metadata interface{}

	Transport *Transport

	responseDispatcherRegistrar *DispatcherTable

	kafkaWriter *kafka.Writer
	logger      *logrus.Entry
}

func (r *ReceptorService) RegisterConnection(peerNodeID string, metadata interface{}) error {
	r.logger.Debugf("Registering a connection to node %s", peerNodeID)

	r.PeerNodeID = peerNodeID
	r.Metadata = metadata

	return nil
}

func (r *ReceptorService) UpdateRoutingTable(edges string, seen string) error {
	r.logger.Debug("edges:", edges)
	r.logger.Debug("seen:", seen)
	return nil
}

func (r *ReceptorService) SendMessage(msgSenderCtx context.Context, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {

	messageID, err := uuid.NewRandom()
	if err != nil {
		r.logger.Println("Unable to generate UUID for routing the job...cannot proceed")
		return nil, err
	}

	payloadMessage, err := protocol.BuildPayloadMessage(
		messageID,
		r.NodeID,
		recipient,
		route,
		"directive",
		directive,
		payload)
	r.logger.Printf("Sending PayloadMessage - %s\n", messageID)

	msgSenderCtx, cancel := context.WithTimeout(msgSenderCtx, time.Second*10) // FIXME:  add a configurable timeout
	defer cancel()

	err = r.sendMessage(msgSenderCtx, payloadMessage)
	if err != nil {
		return nil, err
	}

	return &messageID, nil
}

func (r *ReceptorService) Ping(msgSenderCtx context.Context, recipient string, route []string) (interface{}, error) {

	messageID, err := uuid.NewRandom()
	if err != nil {
		r.logger.Println("Unable to generate UUID for routing the job...cannot proceed")
		return nil, err
	}

	payloadMessage, err := protocol.BuildPayloadMessage(
		messageID,
		r.NodeID,
		recipient,
		route,
		"directive",
		"receptor:ping",
		time.Now().UTC())

	responseChannel := make(chan ResponseMessage)

	r.logger.Println("Registering a sync response handler")
	r.responseDispatcherRegistrar.Register(messageID, responseChannel)
	defer r.responseDispatcherRegistrar.Unregister(messageID)

	msgSenderCtx, cancel := context.WithTimeout(msgSenderCtx, time.Second*10) // FIXME:  add a configurable timeout
	defer cancel()

	err = r.sendControlMessage(msgSenderCtx, payloadMessage)
	if err != nil {
		return nil, err
	}

	responseMsg, err := r.waitForResponse(msgSenderCtx, responseChannel)
	if err != nil {
		return nil, err
	}

	return responseMsg, nil
}

// FIXME:  Does it make sense to move this logic to the transport object?  Or am I missing an abstraction?
func (r *ReceptorService) sendControlMessage(msgSenderCtx context.Context, msgToSend protocol.Message) error {

	return sendMessage(r.logger, r.Transport.Ctx, r.Transport.ControlChannel, msgSenderCtx, msgToSend)
}

func (r *ReceptorService) sendMessage(msgSenderCtx context.Context, msgToSend protocol.Message) error {

	return sendMessage(r.logger, r.Transport.Ctx, r.Transport.Send, msgSenderCtx, msgToSend)
}

func sendMessage(logger *logrus.Entry, transportCtx context.Context, sendChannel chan protocol.Message, msgSenderCtx context.Context, msgToSend protocol.Message) error {
	logger.Println("Passing message to async layer")
	select {

	case sendChannel <- msgToSend:
		return nil

	case <-transportCtx.Done():
		logger.Printf("Connection to receptor network lost")
		return connectionToReceptorNetworkLost

	case <-msgSenderCtx.Done():
		switch msgSenderCtx.Err().(error) {
		case context.DeadlineExceeded:
			logger.Printf("Timed out waiting to pass message to async layer")
			return requestTimedOut
		default:
			logger.Printf("Message cancelled by sender")
			return requestCancelledBySender
		}
	}
}

// FIXME:  Does it make sense to move this logic to the transport object?  Or am I missing an abstraction?
func (r *ReceptorService) waitForResponse(msgSenderCtx context.Context, responseChannel chan ResponseMessage) (ResponseMessage, error) {
	r.logger.Println("Waiting for a sync response")
	nilResponseMessage := ResponseMessage{}

	select {

	case responseMsg := <-responseChannel:
		return responseMsg, nil

	case <-r.Transport.Ctx.Done():
		r.logger.Printf("Connection to receptor network lost")
		return nilResponseMessage, connectionToReceptorNetworkLost

	case <-msgSenderCtx.Done():
		switch msgSenderCtx.Err().(error) {
		case context.DeadlineExceeded:
			r.logger.Printf("Timed out waiting for response for message")
			return nilResponseMessage, requestTimedOut
		default:
			r.logger.Printf("Message cancelled by sender")
			return nilResponseMessage, requestCancelledBySender
		}
	}
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

	inResponseTo, err := uuid.Parse(payloadMessage.Data.InResponseTo)
	if err != nil {
		r.logger.Printf("Unable to convert InResponseTo field into a UUID while dispatching the response.  "+
			"  Error: %s, Message: %+v", err, payloadMessage.Data.InResponseTo)
		return
	}

	responseChannel, _ := r.responseDispatcherRegistrar.GetDispatchChannel(inResponseTo)

	if responseChannel != nil {
		responseChannel <- responseMessage
		return
	}

	r.logger.WithFields(logrus.Fields{"in_response_to": inResponseTo}).Info("Dispatching response message")

	jsonResponseMessage, err := json.Marshal(responseMessage)
	if err != nil {
		r.logger.Println("JSON marshal of ResponseMessage failed, err:", err)
		return
	}

	// FIXME:  spawn a go routine here?  Make sure to honor the ctx
	go func() {
		r.kafkaWriter.WriteMessages(r.Transport.Ctx,
			kafka.Message{
				Key:   []byte(payloadMessage.Data.InResponseTo),
				Value: jsonResponseMessage,
			})
	}()

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

type DispatcherTable struct {
	dispatchTable map[uuid.UUID]chan ResponseMessage
	sync.Mutex
}

func (dt *DispatcherTable) Register(msgID uuid.UUID, responseChannel chan ResponseMessage) {
	dt.Lock()
	dt.dispatchTable[msgID] = responseChannel
	dt.Unlock()
}

func (dt *DispatcherTable) Unregister(msgID uuid.UUID) {
	dt.Lock()
	delete(dt.dispatchTable, msgID)
	dt.Unlock()
}

func (dt *DispatcherTable) GetDispatchChannel(msgID uuid.UUID) (chan ResponseMessage, error) {
	var dispatchChannel chan ResponseMessage

	dt.Lock()
	dispatchChannel, _ = dt.dispatchTable[msgID]
	dt.Unlock()

	return dispatchChannel, nil
}
