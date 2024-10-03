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

package pd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"

	"github.com/PagerDuty/go-pagerduty"
)

type Client interface {
	Setup(ctx context.Context, check config.Check) (*pagerduty.Service, error)
	SetupTrigger(ctx context.Context, check config.Check, service *pagerduty.Service) error

	UpdateMaintenanceWindow(ctx context.Context, maintWindow *pagerduty.MaintenanceWindow, start, end time.Time) error
}

func NewClient(conf *config.PagerDuty) (Client, error) {
	if conf == nil {
		return nil, nil
	}

	cc := &client{
		pdConfig:   *conf,
		underlying: pagerduty.NewClient(conf.ApiKey),
	}
	if err := cc.ping(); err != nil {
		return nil, err
	}

	return cc, nil
}

type client struct {
	pdConfig   config.PagerDuty
	underlying *pagerduty.Client
}

var _ Client = (&client{})

func (c *client) ping() error {
	ctx := context.Background()
	resp, err := c.underlying.ListAbilitiesWithContext(ctx)
	if err != nil {
		return fmt.Errorf("pagerduty list abilities: %v", err)
	}
	if len(resp.Abilities) <= 0 {
		return errors.New("pagerduty: missing abilities")
	}
	return nil
}
