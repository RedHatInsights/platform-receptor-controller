package utils

import (
	"context"
	"net/http"

	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func StartHTTPServer(addr, name string, handler *mux.Router) *http.Server {
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		logger.Log.Infof("Starting %s server:  %s", name, addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Log.WithFields(logrus.Fields{"error": err}).Fatalf("%s server error", name)
		}
	}()

	return srv
}

func ShutdownHTTPServer(ctx context.Context, name string, srv *http.Server) {
	logger.Log.Infof("Shutting down %s server", name)
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Infof("Error shutting down %s server: %e", name, err)
	}
}

func GetHostname() string {
	name, err := os.Hostname()
	if err != nil {
		logger.Log.Info("Error getting hostname")
	}

	return name
}
