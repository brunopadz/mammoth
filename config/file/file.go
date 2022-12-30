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

package file

import (
	"github.com/spf13/viper"

	"github.com/brunopadz/mammoth/util/log"
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetDefault("client.tryssl", true)
}

type ServerConfig struct {
	Cert             string `mapstructure:"cert,omitempty"`
	Key              string `mapstructure:"key,omitempty"`
	CA               string `mapstructure:"ca,omitempty"`
	AllowUnencrypted bool   `mapstructure:"allowunencrypted,omitempty"`
}

type ClientConfig struct {
	Cert             string `mapstructure:"cert,omitempty"`
	Key              string `mapstructure:"key,omitempty"`
	CA               string `mapstructure:"ca,omitempty"`
	SkipVerifyCA     bool   `mapstructure:"skipverify,omitempty"`
	AllowUnencrypted bool   `mapstructure:"allowunencrypted,omitempty"`
	TrySSL           bool   `mapstructure:"tryssl"`
}

type Config struct {
	Bind      string       `mapstructure:"bind"`
	Server    ServerConfig `mapstructure:"server"`
	Client    ClientConfig `mapstructure:"client"`
	HostRegex string       `mapstructure:"hostregex"`
}

func SetConfigPath(path string) {
	viper.SetConfigFile(path)
}

func ReadConfig() (*Config, error) {
	log.Debugf("Reading configuration file: %s", viper.ConfigFileUsed())

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	c := &Config{}

	if err := viper.Unmarshal(c); err != nil {
		log.Errorf("Error unmarshaling configuration file: %s", viper.ConfigFileUsed())
		return nil, err
	}

	return c, nil
}
