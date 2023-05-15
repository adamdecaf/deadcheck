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

package check

import (
	"context"
	"fmt"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/adamdecaf/deadcheck/internal/pd"

	"github.com/moov-io/base/log"
)

type Instances struct {
	pdClient pd.Client
	checks   []config.Check
}

func Setup(logger log.Logger, conf *config.Config) (*Instances, error) {
	pdClient, err := pd.NewClient(conf.PagerDuty)
	if err != nil {
		return nil, fmt.Errorf("setting up base PD client: %w", err)
	}
	for i := range conf.Checks {
		err := pdClient.Setup(conf.Checks[i])
		if err != nil {
			return nil, fmt.Errorf("problem setting up checks[%d]: %w", i, err)
		}
	}

	return &Instances{
		pdClient: pdClient,
		checks:   conf.Checks,
	}, nil
}

func (xs *Instances) CheckIn(ctx context.Context, checkID string) error {
	var found *config.Check
	for i := range xs.checks {
		if xs.checks[i].ID == checkID {
			found = &xs.checks[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("check %s not found", checkID)
	}

	sw := xs.pdClient.ReadSwitch(*found)
	if sw == nil {
		return fmt.Errorf("switch %s not found", found.ID)
	}

	// TODO(adam): adjust MW start/end

	// Need to store/read MW from Switch
	// err = xs.pdClient.UpdateMaintenanceWindow(sw.

	return nil
}
