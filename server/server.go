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

package server

import (
	"net"

	"github.com/brunopadz/mammoth/config"
	"github.com/brunopadz/mammoth/util/log"
)

type Server struct {
	c     *config.Config
	proxy *ProxyServer
}

func NewServer(c *config.Config) *Server {
	s := &Server{
		c:     c,
		proxy: NewProxyServer(c),
	}
	return s
}

func (s *Server) Start() error {
	log.Info("Server starting...")
	proxyListener, err := net.Listen("tcp", s.c.Bind)
	if err != nil {
		log.Fatalf("Could not create listener on %v: %v\n", s.c.Bind, err)
		return err
	}

	s.proxy.Serve(proxyListener)

	log.Info("Server exiting...")
	return nil
}

func (s *Server) Stop() {
	s.proxy.Stop()
}
