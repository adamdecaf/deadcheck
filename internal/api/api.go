// Licensed to Adam Shannon under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. The Moov Authors licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
