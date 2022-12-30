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

package proxy

import (
	"io"
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/brunopadz/mammoth/config"
	"github.com/brunopadz/mammoth/util/log"
)

type Proxy struct {
	Config  *config.Config
	Secrets *BackendSecrets
}

func NewProxy(c *config.Config) *Proxy {
	return &Proxy{
		Config:  c,
		Secrets: NewBackendSecrets(),
	}
}

// HandleConnection handle an incoming connection to the proxy
func (p *Proxy) HandleConnection(conn net.Conn) error {
	l := log.WithFields(logrus.Fields{
		"client": conn.RemoteAddr().String(),
	})
	l.Info("Accepting connection")

	err := (&ProxyConnection{
		c:       p.Config,
		secrets: p.Secrets,
		log:     l,
	}).HandleConnection(conn)

	if err != nil && err != io.EOF {
		l.Infof("Connection handling closed with error: %v", err)
		return err
	}

	l.Info("Connection closed")
	return nil
}
