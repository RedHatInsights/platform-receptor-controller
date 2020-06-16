package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/google/uuid"
)

var (
	errUnableToSendMessage     = errors.New("unable to send message")
	errUnableToProcessResponse = errors.New("unable to process response")
)

type ReceptorHttpProxy struct {
	Hostname      string
	AccountNumber string
	NodeID        string
	Config        *config.Config
}

func (rhp *ReceptorHttpProxy) SendMessage(ctx context.Context, accountNumber string, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {

	probe := createProbe(ctx)

	probe.sendingMessage(accountNumber, recipient)

	postPayload := jobRequest{accountNumber, recipient, payload, directive}
	jsonStr, err := json.Marshal(postPayload)
	if err != nil {
		probe.failedToSendMessage("Unable to send message.  Failed to marshal JSON payload.", err)
		return nil, errUnableToSendMessage
	}

	resp, err := makeHttpRequest(
		ctx,
		http.MethodPost,
		rhp.generateUrl("job"),
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonStr),
	)

	if err != nil {
		probe.failedToSendMessage("Unable to send message.  Failed to create HTTP Request.", err)
		return nil, errUnableToSendMessage
	}

	defer resp.Body.Close()

	jobResponse := jobResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&jobResponse); err != nil {
		probe.failedToProcessMessageResponse("Unable to parse response from receptor-gateway.", err)
		return nil, errUnableToProcessResponse
	}

	messageID, err := uuid.Parse(jobResponse.JobID)
	if err != nil {
		probe.failedToProcessMessageResponse("Unable to read message id from receptor-gateway", err)
		return nil, errUnableToProcessResponse
	}

	probe.messageSent(messageID)

	return &messageID, nil
}

func (rhp *ReceptorHttpProxy) Ping(ctx context.Context, accountNumber string, recipient string, route []string) (interface{}, error) {
	probe := createProbe(ctx)

	probe.sendingPing(accountNumber, recipient)

	postPayload := connectionID{accountNumber, recipient}
	jsonStr, err := json.Marshal(postPayload)
	if err != nil {
		probe.failedToSendPingMessage("Unable to send message.  Failed to marshal JSON payload.", err)
		return nil, errUnableToSendMessage
	}

	resp, err := makeHttpRequest(
		ctx,
		http.MethodPost,
		rhp.generateUrl("connection/ping"),
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonStr),
	)

	if err != nil {
		probe.failedToSendPingMessage("Unable to send message.  Failed to create HTTP Request.", err)
		return nil, errUnableToSendMessage
	}

	defer resp.Body.Close()

	pingResponse := connectionPingResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&pingResponse); err != nil {
		probe.failedToProcessPingMessageResponse("Unable to read response from receptor-gateway", err)
		return nil, errUnableToProcessResponse
	}

	probe.pingMessageSent()

	return pingResponse.Payload, nil
}

func (rhp *ReceptorHttpProxy) Close(ctx context.Context) error {

	probe := createProbe(ctx)

	probe.closingConnection(rhp.AccountNumber, rhp.NodeID)

	postPayload := connectionID{rhp.AccountNumber, rhp.NodeID}
	jsonStr, err := json.Marshal(postPayload)

	if err != nil {
		probe.failedToMarshalJsonPayload(err)
		return errUnableToSendMessage
	}

	resp, err := makeHttpRequest(
		ctx,
		http.MethodPost,
		rhp.generateUrl("connection/disconnect"),
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonStr),
	)

	if err != nil {
		probe.failedToCreateHttpRequest(err)
		return errUnableToSendMessage
	}

	defer resp.Body.Close()

	return nil
}

func (rhp *ReceptorHttpProxy) GetCapabilities(ctx context.Context) (interface{}, error) {
	probe := createProbe(ctx)

	probe.gettingCapabilities(rhp.AccountNumber, rhp.NodeID)

	postPayload := connectionID{rhp.AccountNumber, rhp.NodeID}
	jsonStr, err := json.Marshal(postPayload)
	if err != nil {
		probe.failedToMarshalJsonPayload(err)
		return nil, errUnableToSendMessage
	}

	resp, err := makeHttpRequest(
		ctx,
		http.MethodPost,
		rhp.generateUrl("connection/status"),
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonStr),
	)

	if err != nil {
		probe.failedToCreateHttpRequest(err)
		return nil, errUnableToSendMessage
	}

	defer resp.Body.Close()

	statusResponse := connectionStatusResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&statusResponse); err != nil {
		probe.failedToParseHttpResponse(err)
		return nil, errUnableToProcessResponse
	}

	return statusResponse.Capabilities, nil
}

func (rhp *ReceptorHttpProxy) generateUrl(path string) string {
	return fmt.Sprintf("%s://%s:%d/%s",
		rhp.Config.JobReceiverReceptorProxyScheme,
		rhp.Hostname,
		rhp.Config.JobReceiverReceptorProxyPort,
		path)
}

func makeHttpRequest(ctx context.Context, method, url, accountNumber string, config *config.Config, body io.Reader) (*http.Response, error) {

	ctx, cancel := context.WithTimeout(ctx, config.JobReceiverReceptorProxyTimeout)
	defer cancel()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	addPreSharedKeyHeaders(req.Header, config, accountNumber)

	addRequestIdHeader(req.Header, ctx)

	return http.DefaultClient.Do(req.WithContext(ctx))
}

func addPreSharedKeyHeaders(headers http.Header, config *config.Config, accountNumber string) {
	clientID := config.JobReceiverReceptorProxyClientID
	psk := config.JobReceiverReceptorProxyPSK

	if clientID == "" || psk == "" {
		fmt.Println("[WARN] clientID / psk is nil")
	}

	headers.Set("x-rh-receptor-controller-client-id", clientID)
	headers.Set("x-rh-receptor-controller-account", accountNumber)
	headers.Set("x-rh-receptor-controller-psk", psk)
}

func addRequestIdHeader(headers http.Header, ctx context.Context) {
	requestId := request_id.GetReqID(ctx)
	headers.Set("x-rh-insights-request-id", requestId)
}
