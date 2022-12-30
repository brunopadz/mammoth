/*
Copyright 2017 Crunchy Data Solutions, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cli

import (
	"fmt"
	"os"

	"github.com/brunopadz/mammoth/config"
	"github.com/brunopadz/mammoth/config/file"
	"github.com/brunopadz/mammoth/server"
	"github.com/brunopadz/mammoth/util/log"
	"github.com/spf13/cobra"
)

var configPath string
var logLevel string
var logFormat string

func init() {
	mainCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to configuration file")
	mainCmd.Flags().StringVarP(&logLevel, "log-level", "", "info", "log level")
	mainCmd.Flags().StringVarP(&logFormat, "log-format", "", "plain", "the log output format")
}

var mainCmd = &cobra.Command{
	Use:   "mammoth",
	Short: "A simple Postgres jump server",
	Run:   runStart,
}

func runStart(cmd *cobra.Command, args []string) {
	log.SetLevel(logLevel)
	log.SetFormat(logFormat)

	if configPath != "" {
		file.SetConfigPath(configPath)
	}

	f, err := file.ReadConfig()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
		return
	}

	c, err := config.FromFile(f)
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	s := server.NewServer(c)

	s.Start()

	return
}
func Main() {
	if err := Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Failed running %q\n", os.Args[1])
		os.Exit(1)
	}
}

func Run(args []string) error {
	mainCmd.SetArgs(args)
	return mainCmd.Execute()
}
