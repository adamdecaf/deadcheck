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

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adamdecaf/deadcheck/internal/check"
	"github.com/spf13/viper"
)

type Config struct {
	Checks []check.Config
}

func Load(path string) (*Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("no path specified")
	}

	fullpath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("path %s expansion failed: %v", path, err)
	}

	var cfg Config

	fd, err := os.Open(fullpath)
	if err != nil {
		return nil, err
	}

	reader := viper.New()
	reader.SetConfigType("yaml")
	if err := reader.ReadConfig(fd); err != nil {
		return nil, err
	}
	if err := reader.UnmarshalExact(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}