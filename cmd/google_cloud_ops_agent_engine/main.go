// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/ops-agent/apps"
	"github.com/GoogleCloudPlatform/ops-agent/confgenerator"
	"github.com/GoogleCloudPlatform/ops-agent/internal/healthchecks"
)

var (
	service      = flag.String("service", "", "service to generate config for")
	outDir       = flag.String("out", os.Getenv("RUNTIME_DIRECTORY"), "directory to write configuration files to")
	input        = flag.String("in", "/etc/google-cloud-ops-agent/config.yaml", "path to the user specified agent config")
	logsDir      = flag.String("logs", "/var/log/google-cloud-ops-agent", "path to store agent logs")
	stateDir     = flag.String("state", "/var/lib/google-cloud-ops-agent", "path to store agent state like buffers")
	healthChecks = flag.Bool("healthchecks", false, "run health checks and exit")
)

func runHealthChecks() {
	logger, closer := healthchecks.CreateHealthChecksLogger(*logsDir)
	defer closer()

	healthCheckResults := healthchecks.HealthCheckRegistryFactory().RunAllHealthChecks(logger)
	healthchecks.LogHealthCheckResults(healthCheckResults, func(s string) { log.Println(s) }, func(s string) { log.Println(s) })
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Fatalf("The agent config file is not valid. Detailed error: %s", err)
	}
}

func run() error {
	ctx := context.Background()
	// TODO(lingshi) Move this to a shared place across Linux and Windows.
	uc, err := confgenerator.MergeConfFiles(ctx, *input, apps.BuiltInConfStructs)
	if err != nil {
		return err
	}

	// Log the built-in and merged config files to STDOUT. These are then written
	// by journald to var/log/syslog and so to Cloud Logging once the ops-agent is
	// running.
	log.Printf("Built-in config:\n%s", apps.BuiltInConfStructs["linux"])
	log.Printf("Merged config:\n%s", uc)

	if *service == "" {
		runHealthChecks()
		log.Println("Startup checks finished")
		if *healthChecks {
			// If healthchecks is set, stop here
			return nil
		}
	}
	return uc.GenerateFilesFromConfig(ctx, *service, *logsDir, *stateDir, *outDir)
}
