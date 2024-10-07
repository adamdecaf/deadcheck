package api

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"github.com/adamdecaf/deadcheck/internal/check"

	"github.com/gorilla/mux"
	"github.com/moov-io/base/log"
)

func Server(logger log.Logger, httpAddr string, instances *check.Instances) (*http.Server, error) {
	router := mux.NewRouter()
	serve := &http.Server{
		Addr:    httpAddr,
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
		Methods("PUT").
		Path("/checks/{checkID}/check-in").
		HandlerFunc(checkIn(logger, instances))

	go func() {
		logger.Info().Logf("HTTP server starting on %s", httpAddr)

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
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(checkInResponse{
			NextExpectedCheckIn: resp.NextExpectedCheckIn,
		})
	}
}
