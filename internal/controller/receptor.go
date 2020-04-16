package controller

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/receptor/protocol"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	kafka "github.com/segmentio/kafka-go"

	"github.com/sirupsen/logrus"
)

var (
	connectionToReceptorNetworkLost = errors.New("Connection to receptor network lost")
	requestCancelledBySender        = errors.New("Unable to complete the request.  Request cancelled by message sender.")
	requestTimedOut                 = errors.New("Unable to complete the request.  Request timed out.")
	accountMismatch                 = errors.New("Account mismatch.  Unable to complete the request.")
)

type ReceptorServiceFactory struct {
	kafkaWriter *kafka.Writer
	config      *config.Config
}

func NewReceptorServiceFactory(w *kafka.Writer, cfg *config.Config) *ReceptorServiceFactory {
	return &ReceptorServiceFactory{
		kafkaWriter: w,
		config:      cfg,
	}
}

func (fact *ReceptorServiceFactory) NewReceptorService(logger *logrus.Entry, account, nodeID string) *ReceptorService {
	return &ReceptorService{
		AccountNumber: account,
		NodeID:        nodeID,
		responseDispatcherRegistrar: &DispatcherTable{
			dispatchTable: make(map[uuid.UUID]chan ResponseMessage),
		},
		kafkaWriter: fact.kafkaWriter,
		config:      fact.config,
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
	config      *config.Config
	logger      *logrus.Entry
}

func (r *ReceptorService) RegisterConnection(peerNodeID string, metadata interface{}, transport *Transport) error {
	r.logger.Info("Registering a connection to node ", peerNodeID)

	r.PeerNodeID = peerNodeID
	r.Metadata = metadata
	r.Transport = transport

	return nil
}

func (r *ReceptorService) UpdateRoutingTable(edges string, seen string) error {
	r.logger.Debug("edges:", edges)
	r.logger.Debug("seen:", seen)
	return nil
}

func (r *ReceptorService) SendMessage(msgSenderCtx context.Context, account string, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {

	if account != r.AccountNumber {
		return nil, accountMismatch
	}

	messageID, err := uuid.NewRandom()
	if err != nil {
		r.logger.Info("Unable to generate UUID for routing the job...cannot proceed")
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
	r.logger.Infof("Sending PayloadMessage - %s\n", messageID)

	msgSenderCtx, cancel := context.WithTimeout(msgSenderCtx, r.config.ReceptorSyncPingTimeout)
	defer cancel()

	err = r.sendMessage(msgSenderCtx, payloadMessage)
	if err != nil {
		return nil, err
	}

	return &messageID, nil
}

func (r *ReceptorService) Ping(msgSenderCtx context.Context, account string, recipient string, route []string) (interface{}, error) {

	if account != r.AccountNumber {
		return nil, accountMismatch
	}

	messageID, err := uuid.NewRandom()
	if err != nil {
		r.logger.Info("Unable to generate UUID for routing the job...cannot proceed")
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

	r.logger.Info("Registering a sync response handler")
	r.responseDispatcherRegistrar.Register(messageID, responseChannel)
	defer r.responseDispatcherRegistrar.Unregister(messageID)

	msgSenderCtx, cancel := context.WithTimeout(msgSenderCtx, r.config.ReceptorSyncPingTimeout)
	defer cancel()

	pingDurationRecorder := DurationRecorder{elapsed: metrics.pingElapsed,
		labels: prometheus.Labels{"account": r.AccountNumber, "recipient": r.PeerNodeID}}
	pingDurationRecorder.Start()

	err = r.sendControlMessage(msgSenderCtx, payloadMessage)
	if err != nil {
		return nil, err
	}

	responseMsg, err := r.waitForResponse(msgSenderCtx, responseChannel)
	pingDurationRecorder.Stop()
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
	logger.Debug("Passing message to async layer")

	select {

	case sendChannel <- msgToSend:
		return nil

	case <-transportCtx.Done():
		logger.Info("Connection to receptor network lost")
		return connectionToReceptorNetworkLost

	case <-msgSenderCtx.Done():
		switch msgSenderCtx.Err().(error) {
		case context.DeadlineExceeded:
			logger.Info("Timed out waiting to pass message to async layer")
			return requestTimedOut
		default:
			logger.Info("Message cancelled by sender")
			return requestCancelledBySender
		}
	}
}

// FIXME:  Does it make sense to move this logic to the transport object?  Or am I missing an abstraction?
func (r *ReceptorService) waitForResponse(msgSenderCtx context.Context, responseChannel chan ResponseMessage) (ResponseMessage, error) {
	r.logger.Info("Waiting for a sync response")
	nilResponseMessage := ResponseMessage{}

	select {

	case responseMsg := <-responseChannel:
		return responseMsg, nil

	case <-r.Transport.Ctx.Done():
		r.logger.Info("Connection to receptor network lost")
		return nilResponseMessage, connectionToReceptorNetworkLost

	case <-msgSenderCtx.Done():
		switch msgSenderCtx.Err().(error) {
		case context.DeadlineExceeded:
			r.logger.Info("Timed out waiting for response for message")
			return nilResponseMessage, requestTimedOut
		default:
			r.logger.Info("Message cancelled by sender")
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
		r.logger.Infof("Unable to convert InResponseTo field into a UUID while dispatching the response.  "+
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
		r.logger.Info("JSON marshal of ResponseMessage failed, err:", err)
		return
	}

	// FIXME:  spawn a go routine here?  Make sure to honor the ctx
	go func() {
		metrics.responseKafkaWriterGoRoutineGauge.Inc()
		err = r.kafkaWriter.WriteMessages(r.Transport.Ctx,
			kafka.Message{
				Key:   []byte(payloadMessage.Data.InResponseTo),
				Value: jsonResponseMessage,
			})

		if err != nil {
			r.logger.WithFields(logrus.Fields{"error": err}).Warn("Error writing response message to kafka")
			metrics.responseKafkaWriterFailureCounter.Inc()
		}

		metrics.responseKafkaWriterGoRoutineGauge.Dec()
	}()

}

func (r *ReceptorService) Close() {
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
