package api

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type ApiSpecServer struct {
	router *mux.Router
}

func NewApiSpecServer(r *mux.Router) *ApiSpecServer {
	return &ApiSpecServer{
		router: r,
	}
}

func (s *ApiSpecServer) Routes() {
	s.router.HandleFunc("/openapi.json", s.handleApiSpec()).Methods("GET")
}

func (s *ApiSpecServer) handleApiSpec() http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		apiSpecFileName := "/opt/apt-root/src/apispec/api.spec.json"
		file, err := ioutil.ReadFile(apiSpecFileName)
		if err != nil {
			log.Printf("Unable to read API spec file (%s): %s", apiSpecFileName, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(file)
	}
}
