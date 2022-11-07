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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/moov-io/base/log"
)

var (
	flagConfig = flag.String("config", "", "Filepath to configuration file")

	flagVersion = flag.Bool("version", false, "Print the version of csvq")
)

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Printf("deadcheck %s", Version)
		return
	}

	logger := log.NewDefaultLogger().With(log.Fields{
		"app":     log.String("deadcheck"),
		"version": log.String(Version),
	})

	conf, err := config.Load(*flagConfig)
	if err != nil {
		logger.Error().LogErrorf("reading %s failed: %v", *flagConfig, err)
		os.Exit(1)
	}

	fmt.Printf("%v\n", conf)

	fmt.Println("running")
}
