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
	"time"

	"github.com/adamdecaf/deadcheck/internal/pd"
)

type Config struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	Schedule ScheduleConfig `yaml:schedule"`

	PagerDuty *pd.Config `yaml:"pagerduty"`
}

type ScheduleConfig struct {
	Every       *time.Duration `yaml:"duration"`
	Weekdays    PartialDay     `yaml:"weekdays"`
	BankingDays PartialDay     `yaml:"bankingDays"`
}

type PartialDay struct {
	Timezone string   `yaml:"timezone"`
	Times    []string `yaml:"times"`
}
