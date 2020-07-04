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
	"github.com/spf13/cobra"

	"github.com/twooster/pg-jump/config"
	"github.com/twooster/pg-jump/server"
	"github.com/twooster/pg-jump/util/log"
)

var configPath string
var logLevel string

var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "start a proxy instance",
	Long:    "",
	Example: "",
	RunE:    runStart,
}

func init() {
	flags := startCmd.Flags()
	stringFlag(flags, &configPath, FlagConfigPath)
	stringFlag(flags, &logLevel, FlagLogLevel)
}

func runStart(cmd *cobra.Command, args []string) error {
	log.SetLevel(logLevel)

	if configPath != "" {
		config.SetConfigPath(configPath)
	}

	c, err := config.ReadConfig()
	if err != nil {
		return err
	}

	s := server.NewServer(c)

	s.Start()

	return nil
}
