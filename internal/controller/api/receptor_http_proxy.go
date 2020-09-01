package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/redhatinsights/platform-go-middlewares/request_id"

	"github.com/google/uuid"
)

var (
	errUnableToSendMessage     = errors.New("unable to send message")
	errUnableToProcessResponse = errors.New("unable to process response")
	errDisconnectedNode        = errors.New("disconnected node")
)

type ReceptorHttpProxy struct {
	Hostname      string
	AccountNumber string
	NodeID        string
	Config        *config.Config
}

func (rhp *ReceptorHttpProxy) SendMessage(ctx context.Context, accountNumber string, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {

	probe := createProbe(ctx, "send_message")

	probe.sendingMessage(accountNumber, recipient)

	jsonBytes, err := marshalJobRequest(accountNumber, recipient, payload, directive, probe)
	if err != nil {
		return nil, err
	}

	resp, err := makeHttpRequest(
		ctx,
		probe,
		http.MethodPost,
		rhp.generateUrl("job"),
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonBytes),
	)

	if err != nil {
		probe.failedToMakeHttpRequest(err)
		return nil, errUnableToSendMessage
	}

	defer resp.Body.Close()

	probe.recordHttpStatusCode(resp.StatusCode)

	jobResponse, err := unmarshalJobResponse(resp, probe)
	if err != nil {
		return nil, err
	}

	messageID, err := uuid.Parse(jobResponse.JobID)
	if err != nil {
		probe.failedToUnmarshalResponse(err)
		return nil, errUnableToProcessResponse
	}

	probe.messageSent(messageID)

	return &messageID, nil
}

func (rhp *ReceptorHttpProxy) Ping(ctx context.Context, accountNumber string, recipient string, route []string) (interface{}, error) {
	probe := createProbe(ctx, "ping")

	probe.sendingPing(accountNumber, recipient)

	jsonBytes, err := marshalConnectionKey(accountNumber, recipient, probe)
	if err != nil {
		probe.failedToMarshalPayload(err)
		return nil, err
	}

	resp, err := makeHttpRequest(
		ctx,
		probe,
		http.MethodPost,
		rhp.generateUrl("connection/ping"),
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonBytes),
	)

	if err != nil {
		probe.failedToMakeHttpRequest(err)
		return nil, errUnableToSendMessage
	}

	defer resp.Body.Close()

	probe.recordHttpStatusCode(resp.StatusCode)

	pingResponse, err := unmarshalConnectionPingResponse(resp, probe)
	if err != nil {
		return nil, err
	}

	probe.pingMessageSent()

	if pingResponse.Status == DISCONNECTED_STATUS {
		return nil, errDisconnectedNode
	}

	return pingResponse.Payload, nil
}

func (rhp *ReceptorHttpProxy) Close(ctx context.Context) error {

	probe := createProbe(ctx, "close_connection")

	probe.closingConnection(rhp.AccountNumber, rhp.NodeID)

	jsonBytes, err := marshalConnectionKey(rhp.AccountNumber, rhp.NodeID, probe)
	if err != nil {
		probe.failedToMarshalPayload(err)
		return err
	}

	resp, err := makeHttpRequest(
		ctx,
		probe,
		http.MethodPost,
		rhp.generateUrl("connection/disconnect"),
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonBytes),
	)

	if err != nil {
		probe.failedToMakeHttpRequest(err)
		return errUnableToSendMessage
	}

	defer resp.Body.Close()

	probe.recordHttpStatusCode(resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		probe.invalidHttpStatusCode(resp.StatusCode)
		return nil
	}

	probe.connectionClosed(rhp.AccountNumber, rhp.NodeID)

	return nil
}

func (rhp *ReceptorHttpProxy) GetCapabilities(ctx context.Context) (interface{}, error) {
	probe := createProbe(ctx, "get_capabilities")

	probe.gettingCapabilities(rhp.AccountNumber, rhp.NodeID)

	jsonBytes, err := marshalConnectionKey(rhp.AccountNumber, rhp.NodeID, probe)
	if err != nil {
		probe.failedToMarshalPayload(err)
		return nil, err
	}

	resp, err := makeHttpRequest(
		ctx,
		probe,
		http.MethodPost,
		rhp.generateUrl("connection/status"),
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonBytes),
	)

	if err != nil {
		probe.failedToMakeHttpRequest(err)
		return nil, errUnableToSendMessage
	}

	defer resp.Body.Close()

	probe.recordHttpStatusCode(resp.StatusCode)

	statusResponse, err := unmarshalConnectionStatusResponse(resp, probe)
	if err != nil {
		return nil, err
	}

	probe.retrievedCapabilities(rhp.AccountNumber, rhp.NodeID)

	if statusResponse.Status == DISCONNECTED_STATUS {
		return nil, errDisconnectedNode
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

func marshalJobRequest(accountNumber, recipient string, payload interface{}, directive string, probe *receptorHttpProxyProbe) ([]byte, error) {
	postPayload := jobRequest{accountNumber, recipient, payload, directive}
	jsonBytes, err := json.Marshal(postPayload)
	if err != nil {
		probe.failedToMarshalPayload(err)
		return nil, errUnableToSendMessage
	}

	return jsonBytes, nil
}

func unmarshalJobResponse(resp *http.Response, probe *receptorHttpProxyProbe) (*jobResponse, error) {

	jobResponse := jobResponse{}
	if resp.StatusCode != http.StatusCreated {
		probe.invalidHttpStatusCode(resp.StatusCode)
		if resp.StatusCode == http.StatusNotFound {
			return nil, errDisconnectedNode
		}
		return nil, errUnableToProcessResponse
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&jobResponse); err != nil {
		probe.failedToUnmarshalResponse(err)
		return nil, errUnableToProcessResponse
	}

	return &jobResponse, nil
}

func unmarshalConnectionPingResponse(resp *http.Response, probe *receptorHttpProxyProbe) (*connectionPingResponse, error) {
	if resp.StatusCode != http.StatusOK {
		probe.invalidHttpStatusCode(resp.StatusCode)
		return nil, errUnableToProcessResponse
	}

	pingResponse := connectionPingResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&pingResponse); err != nil {
		probe.failedToUnmarshalResponse(err)
		return nil, errUnableToProcessResponse
	}

	return &pingResponse, nil
}

func unmarshalConnectionStatusResponse(resp *http.Response, probe *receptorHttpProxyProbe) (*connectionStatusResponse, error) {
	if resp.StatusCode != http.StatusOK {
		probe.invalidHttpStatusCode(resp.StatusCode)
		return nil, errUnableToProcessResponse
	}

	statusResponse := connectionStatusResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&statusResponse); err != nil {
		probe.failedToUnmarshalResponse(err)
		return nil, errUnableToProcessResponse
	}

	return &statusResponse, nil
}

func marshalConnectionKey(accountNumber, recipient string, probe *receptorHttpProxyProbe) ([]byte, error) {
	postPayload := connectionID{accountNumber, recipient}
	jsonBytes, err := json.Marshal(postPayload)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func makeHttpRequest(ctx context.Context, probe *receptorHttpProxyProbe, method, url, accountNumber string, config *config.Config, body io.Reader) (*http.Response, error) {

	ctx, cancel := context.WithTimeout(ctx, config.JobReceiverReceptorProxyTimeout)
	defer cancel()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	addPreSharedKeyHeaders(req.Header, config, accountNumber)

	addRequestIdHeader(req.Header, ctx)

	startTime := time.Now()
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	elapsedTime := time.Since(startTime)
	probe.recordRemoteCallDuration(elapsedTime)

	return resp, err
}

func addPreSharedKeyHeaders(headers http.Header, config *config.Config, accountNumber string) {
	clientID := config.JobReceiverReceptorProxyClientID
	psk := config.JobReceiverReceptorProxyPSK

	headers.Set("x-rh-receptor-controller-client-id", clientID)
	headers.Set("x-rh-receptor-controller-account", accountNumber)
	headers.Set("x-rh-receptor-controller-psk", psk)
}

func addRequestIdHeader(headers http.Header, ctx context.Context) {
	requestId := request_id.GetReqID(ctx)
	headers.Set("x-rh-insights-request-id", requestId)
}
