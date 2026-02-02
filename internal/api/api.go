package api

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"github.com/adamdecaf/deadcheck/internal/check"
	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/gorilla/mux"
	"github.com/moov-io/base/log"
)

func Server(logger log.Logger, conf config.ServerConfig, instances *check.Instances) (*http.Server, error) {
	router := mux.NewRouter()
	serve := &http.Server{
		Addr:    conf.BindAddress,
		Handler: router,
		TLSConfig: &tls.Config{
			InsecureSkipVerify:       false,
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS12,
		},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	router.
		Methods("POST", "PUT").
		Path("/checks/{checkID}/check-in").
		HandlerFunc(checkIn(logger, instances))

	go func() {
		logger.Info().Logf("HTTP server starting on %s", conf.BindAddress)

		err := serve.ListenAndServe()
		if err != nil {
			logger.Warn().Logf("http server: %v", err)
		}
	}()

	return serve, nil
}

type checkInResponse struct {
	NextExpectedCheckIn time.Time `json:"nextExpectedCheckIn"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func checkIn(logger log.Logger, instances *check.Instances) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checkID := mux.Vars(r)["checkID"]

		logger := logger.With(log.Fields{
			"check_id": log.String(checkID),
		})
		logger.Log("handling check-in")

		resp, err := instances.CheckIn(r.Context(), logger, checkID)
		if err != nil {
			logger.LogErrorf("problem during check-in: %v", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)

			json.NewEncoder(w).Encode(errorResponse{
				Error: err.Error(),
			})

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(checkInResponse{
			NextExpectedCheckIn: resp.NextExpectedCheckIn,
		})
	}
}
