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

package config

import (
	"github.com/spf13/viper"

	"github.com/twooster/pg-jump/util/log"
)

func init() {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
}

type SSLConfig struct {
	Enable        bool   `mapstructure:"enable"`
	SSLMode       string `mapstructure:"sslmode"`
	SSLCert       string `mapstructure:"sslcert,omitempty"`
	SSLKey        string `mapstructure:"sslkey,omitempty"`
	SSLRootCA     string `mapstructure:"sslrootca,omitempty"`
	SSLServerCert string `mapstructure:"sslservercert,omitempty"`
	SSLServerKey  string `mapstructure:"sslserverkey,omitempty"`
	SSLServerCA   string `mapstructure:"sslserverca,omitempty"`
}

type Config struct {
	HostPort  string    `mapstructure:"hostport"`
	SSLConfig SSLConfig `mapstructure:"ssl"`
}

func Get(key string) interface{} {
	return viper.Get(key)
}

func GetBool(key string) bool {
	return viper.GetBool(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetStringMapString(key string) map[string]string {
	return viper.GetStringMapString(key)
}

func GetStringMap(key string) map[string]interface{} {
	return viper.GetStringMap(key)
}

func GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}

func IsSet(key string) bool {
	return viper.IsSet(key)
}

func Set(key string, value interface{}) {
	viper.Set(key, value)
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
