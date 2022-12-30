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
	"github.com/brunopadz/mammoth/proxy"
	"github.com/brunopadz/mammoth/util/log"
)

type ProxyServer struct {
	ch       chan bool
	p        *proxy.Proxy
	listener net.Listener
}

func NewProxyServer(c *config.Config) *ProxyServer {
	p := &ProxyServer{
		ch: make(chan bool),
		p:  proxy.NewProxy(c),
	}

	return p
}

func (s *ProxyServer) Serve(l net.Listener) {
	log.Infof("Proxy Server listening on: %s", l.Addr())

	s.listener = l

	for {
		select {
		case <-s.ch:
			return
		default:
		}

		conn, err := l.Accept()
		if err != nil {
			continue
		}

		go s.p.HandleConnection(conn)
	}
}

func (s *ProxyServer) Stop() {
	s.listener.Close()
	close(s.ch)
}
