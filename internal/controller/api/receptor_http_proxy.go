package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"

	"github.com/google/uuid"
)

type ReceptorHttpProxy struct {
	Url           string
	AccountNumber string
	NodeID        string
	ClientID      string
	PSK           string
}

func (rhp *ReceptorHttpProxy) SendMessage(ctx context.Context, accountNumber string, recipient string, route []string, payload interface{}, directive string) (*uuid.UUID, error) {
	logger.Log.Printf("SendMessage")

	postPayload := jobRequest{accountNumber, recipient, payload, directive}
	jsonStr, err := json.Marshal(postPayload)
	logger.Log.Printf("jsonStr: %s", jsonStr)

	req, err := http.NewRequest(http.MethodPost, rhp.Url+"/job", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rh-receptor-controller-client-id", rhp.ClientID)
	req.Header.Set("x-rh-receptor-controller-account", rhp.AccountNumber)
	req.Header.Set("x-rh-receptor-controller-psk", rhp.PSK)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
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

	req, err := http.NewRequest(http.MethodPost, rhp.Url+"/connection/ping", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rh-receptor-controller-client-id", rhp.ClientID)
	req.Header.Set("x-rh-receptor-controller-account", rhp.AccountNumber)
	req.Header.Set("x-rh-receptor-controller-psk", rhp.PSK)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
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

func (rhp *ReceptorHttpProxy) Close() {
	logger.Log.Printf("Close")

	postPayload := connectionID{rhp.AccountNumber, rhp.NodeID}
	jsonStr, err := json.Marshal(postPayload)
	logger.Log.Printf("jsonStr: %s", jsonStr)

	req, err := http.NewRequest(http.MethodPost, rhp.Url+"/connection/disconnect", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rh-receptor-controller-client-id", rhp.ClientID)
	req.Header.Set("x-rh-receptor-controller-account", rhp.AccountNumber)
	req.Header.Set("x-rh-receptor-controller-psk", rhp.PSK)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Printf("ERROR:%s\n", err)
		// FIXME:
	}

	defer resp.Body.Close()

	return
}

func (rhp *ReceptorHttpProxy) GetCapabilities() interface{} {
	logger.Log.Printf("GetCapabilities")
	return nil
}
