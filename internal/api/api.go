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
	"net/http"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/moov-io/base/log"
)

func Server(logger log.Logger, conf config.Config) (*http.Server, error) {
	// TODO(adam): mux Router

	// PUT /checks/{id}/check-in

	// TODO(adam): endpoint check-in extends maint window
	// func (c *client) updateMaintenanceWindow(maintWindow *pagerduty.MaintenanceWindow, start, end time.Time) error

	return nil, nil
}
