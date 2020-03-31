package api

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorResponse struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func WriteJSONResponse(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Unable to encode payload!", http.StatusUnprocessableEntity)
		log.Println("Unable to encode payload!")
		return
	}
}
