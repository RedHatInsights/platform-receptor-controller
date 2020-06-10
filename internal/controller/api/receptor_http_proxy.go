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

	"github.com/google/uuid"
)

// FIXME:
//   - request_id
//   - timeouts
//   - improve error reporting
//   - verify account number and recipient

type ReceptorHttpProxy struct {
	Url           string
	AccountNumber string
	NodeID        string
	Config        *config.Config
}

func makeHttpRequest(ctx context.Context, method, url, accountNumber string, config *config.Config, body io.Reader) (*http.Response, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	clientID := config.JobReceiverClientID
	psk := config.JobReceiverPSK

	if clientID == "" || psk == "" {
		fmt.Println("[WARN] clientID / psk is nil")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rh-receptor-controller-client-id", clientID)
	req.Header.Set("x-rh-receptor-controller-account", accountNumber)
	req.Header.Set("x-rh-receptor-controller-psk", psk)

	// FIXME: add request_id

	return http.DefaultClient.Do(req.WithContext(ctx))
}

func (rhp *ReceptorHttpProxy) SendMessage(ctx context.Context, accountNumber string, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {
	sendingMessage(accountNumber, recipient)

	postPayload := jobRequest{accountNumber, recipient, payload, directive}
	jsonStr, err := json.Marshal(postPayload)
	if err != nil {
		failedToMarshalJsonPayload(err)
		// FIXME:  SPECIFIC error message
		return nil, err
	}

	resp, err := makeHttpRequest(
		ctx,
		http.MethodPost,
		rhp.Url+"/job",
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonStr),
	)

	if err != nil {
		failedToCreateHttpRequest(err)
		// FIXME:  SPECIFIC error message
		return nil, err
	}

	defer resp.Body.Close()

	jobResponse := jobResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&jobResponse); err != nil {
		failedToParseHttpResponse(err)
		// FIXME:  SPECIFIC error message
		return nil, err // errors.New(errMsg)
	}

	messageID, err := uuid.Parse(jobResponse.JobID)
	if err != nil {
		failedToParseMessageID(err)
		// FIXME:  SPECIFIC error message
		return nil, err // errors.New(errMsg)
	}

	return &messageID, nil
}

func (rhp *ReceptorHttpProxy) Ping(ctx context.Context, accountNumber string, recipient string, route []string) (interface{}, error) {
	pingingNode(accountNumber, recipient)

	postPayload := connectionID{accountNumber, recipient}
	jsonStr, err := json.Marshal(postPayload)
	if err != nil {
		failedToMarshalJsonPayload(err)
		// FIXME:  SPECIFIC error message
		return nil, err
	}

	resp, err := makeHttpRequest(
		ctx,
		http.MethodPost,
		rhp.Url+"/connection/ping",
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonStr),
	)

	if err != nil {
		failedToCreateHttpRequest(err)
		// FIXME:
		return nil, err
	}

	defer resp.Body.Close()

	pingResponse := connectionPingResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&pingResponse); err != nil {
		failedToParseHttpResponse(err)
		return nil, errors.New("Unable to read response from receptor-gateway")
	}

	return pingResponse.Payload, nil
}

func (rhp *ReceptorHttpProxy) Close(ctx context.Context) error {

	closingConnection(rhp.AccountNumber, rhp.NodeID)

	postPayload := connectionID{rhp.AccountNumber, rhp.NodeID}
	jsonStr, err := json.Marshal(postPayload)

	if err != nil {
		failedToMarshalJsonPayload(err)
		// FIXME:  SPECIFIC error message
		return err
	}

	resp, err := makeHttpRequest(
		ctx,
		http.MethodPost,
		rhp.Url+"/connection/disconnect",
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonStr),
	)

	if err != nil {
		failedToCreateHttpRequest(err)
		// FIXME:  SPECIFIC error message
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (rhp *ReceptorHttpProxy) GetCapabilities(ctx context.Context) (interface{}, error) {
	gettingCapabilities(rhp.AccountNumber, rhp.NodeID)

	postPayload := connectionID{rhp.AccountNumber, rhp.NodeID}
	jsonStr, err := json.Marshal(postPayload)
	if err != nil {
		failedToMarshalJsonPayload(err)
		// FIXME:  SPECIFIC error message
		return nil, err
	}

	resp, err := makeHttpRequest(
		ctx,
		http.MethodPost,
		rhp.Url+"/connection/status",
		rhp.AccountNumber,
		rhp.Config,
		bytes.NewBuffer(jsonStr),
	)

	if err != nil {
		failedToCreateHttpRequest(err)
		// FIXME:  SPECIFIC error message
		return nil, err
	}

	defer resp.Body.Close()

	statusResponse := connectionStatusResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&statusResponse); err != nil {
		failedToParseHttpResponse(err)
		// FIXME:  SPECIFIC error message
		return nil, err
	}

	// FIXME: return an error
	return statusResponse.Capabilities, nil
}
