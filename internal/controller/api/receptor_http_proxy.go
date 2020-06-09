package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/google/uuid"
)

// FIXME:
//   - request_id
//   - timeouts
//   - improve error reporting

type ReceptorHttpProxy struct {
	Url           string
	AccountNumber string
	NodeID        string
	ClientID      string
	PSK           string
}

func makeHttpRequest(method, url, clientID, accountNumber, psk string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	fmt.Println("clientID:", clientID)
	fmt.Println("accountNumber:", accountNumber)
	fmt.Println("psk:", psk)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rh-receptor-controller-client-id", clientID)
	req.Header.Set("x-rh-receptor-controller-account", accountNumber)
	req.Header.Set("x-rh-receptor-controller-psk", psk)

	// FIXME: add request_id

	client := &http.Client{}

	return client.Do(req)
}

func (rhp *ReceptorHttpProxy) SendMessage(ctx context.Context, accountNumber string, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {
	logger.Log.Printf("SendMessage")

	postPayload := jobRequest{accountNumber, recipient, payload, directive}
	jsonStr, err := json.Marshal(postPayload)
	logger.Log.Printf("jsonStr: %s", jsonStr)

	resp, err := makeHttpRequest(http.MethodPost, rhp.Url+"/job", rhp.ClientID, rhp.AccountNumber, rhp.PSK, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Println("ERROR: ", err)
		// FIXME:
		return nil, err
	}

	defer resp.Body.Close()

	jobResponse := jobResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&jobResponse); err != nil {
		logger.Log.Error("Unable to read response from receptor-gateway")
		return nil, errors.New("Unable to read response from receptor-gateway")
	}

	messageID, err := uuid.Parse(jobResponse.JobID)
	if err != nil {
		logger.Log.Error("Unable to read message id from receptor-gateway")
		return nil, errors.New("Unable to read response from receptor-gateway")
	}

	return &messageID, nil
}

func (rhp *ReceptorHttpProxy) Ping(ctx context.Context, accountNumber string, recipient string, route []string) (interface{}, error) {
	logger.Log.Printf("Ping")

	postPayload := connectionID{accountNumber, recipient}
	jsonStr, err := json.Marshal(postPayload)
	logger.Log.Printf("jsonStr: %s", jsonStr)

	resp, err := makeHttpRequest(http.MethodPost, rhp.Url+"/connection/ping", rhp.ClientID, rhp.AccountNumber, rhp.PSK, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Println("ERROR: ", err)
		// FIXME:
		return nil, err
	}

	defer resp.Body.Close()

	pingResponse := connectionPingResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&pingResponse); err != nil {
		logger.Log.Error("Unable to read response from receptor-gateway")
		return nil, errors.New("Unable to read response from receptor-gateway")
	}

	return pingResponse.Payload, nil
}

func (rhp *ReceptorHttpProxy) Close(ctx context.Context) error {
	logger.Log.Printf("Close")

	postPayload := connectionID{rhp.AccountNumber, rhp.NodeID}
	jsonStr, err := json.Marshal(postPayload)
	logger.Log.Printf("jsonStr: %s", jsonStr)

	resp, err := makeHttpRequest(
		http.MethodPost,
		rhp.Url+"/connection/disconnect",
		rhp.ClientID, rhp.AccountNumber, rhp.PSK,
		bytes.NewBuffer(jsonStr))

	if err != nil {
		logger.Log.Printf("ERROR:%s\n", err)
		// FIXME:
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (rhp *ReceptorHttpProxy) GetCapabilities(ctx context.Context) (interface{}, error) {
	logger.Log.Printf("GetCapabilities")
	postPayload := connectionID{rhp.AccountNumber, rhp.NodeID}
	jsonStr, err := json.Marshal(postPayload)
	logger.Log.Printf("jsonStr: %s", jsonStr)

	resp, err := makeHttpRequest(
		http.MethodPost,
		rhp.Url+"/connection/status",
		rhp.ClientID, rhp.AccountNumber, rhp.PSK,
		bytes.NewBuffer(jsonStr))

	if err != nil {
		logger.Log.Println("ERROR: ", err)
		// FIXME: log and return an error
		return nil, err
	}

	defer resp.Body.Close()

	statusResponse := connectionStatusResponse{}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&statusResponse); err != nil {
		logger.Log.Error("Unable to read response from receptor-gateway")
		return nil, err
	}

	// FIXME: return an error
	return statusResponse.Capabilities, nil
}
