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
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/twooster/pg-jump/config"
	"github.com/twooster/pg-jump/protocol"
	"github.com/twooster/pg-jump/util/log"
)

type Proxy struct {
	SSLConfig *config.SSLConfig
}

func parseStartupMessage(r *protocol.Reader) (hostPort string, newStartupMessage *protocol.Buffer, e error) {
	var database string
	props := map[string]string{}

	for {
		key, err := r.ReadString()
		if err != nil {
			e = err
			return
		}

		// startupmessage is terminated by a 0x00 byte, which
		// will be parsed by an empty string key
		if key == "" {
			if err = r.Finalize(); err != nil {
				e = err
				return
			}
			break
		}

		val, err := r.ReadString()
		if err != nil {
			e = err
			return
		}

		if key == "database" {
			database = val
		} else {
			props[key] = val
		}
	}

	// parse database: host:port/database
	var split []string
	split = strings.SplitN(database, "/", 2)
	if len(split) != 2 {
		return "", nil, errors.New("Database string missing /")
	}
	hostPort = split[0]
	props["database"] = split[1]

	newStartupMessage = protocol.NewBuffer()
	newStartupMessage.WriteInt32(protocol.ProtocolVersion)
	for k, v := range props {
		newStartupMessage.WriteString(k)
		newStartupMessage.WriteString(v)
	}
	newStartupMessage.WriteByte(0x00)
	return
}

// HandleConnection handle an incoming connection to the proxy
func (p *Proxy) HandleConnection(clientConn net.Conn) error {
	log.Infof("Accepting connection from %v", clientConn.RemoteAddr())

	r, err := protocol.ReadMessage(clientConn)
	if err != nil {
		return fmt.Errorf("Error reading initial StartupMessage: %w", err)
	}

	version, err := r.ReadInt32()
	if err != nil {
		return fmt.Errorf("Error reading initial StartupMessage: %w", err)
	}

	if version == protocol.SSLRequestCode {
		log.Infof("User requesting SSL upgrade")

		if err := r.Finalize(); err != nil {
			return fmt.Errorf("Error while upgrading to SSL connection: %w", err)
		}

		/* Determine which SSL response to send to client. */
		if p.SSLConfig.Enable {
			log.Info("Upgrading SSL connection")
			clientConn.Write([]byte{protocol.SSLAllowed})
			/* Upgrade the client connection if required. */
			clientConn = UpgradeServerConnection(clientConn, p.SSLConfig)
		} else {
			log.Info("Rejecting SSL handshake")
			_, err := clientConn.Write([]byte{protocol.SSLNotAllowed})
			if err != nil {
				return fmt.Errorf("Error during SSL upgrade: %v", err)
			}
		}

		/*
		 * Re-read the startup message from the client. It is possible that the
		 * client might not like the response given and as a result it might
		 * close the connection. This is not an 'error' condition as this is an
		 * expected behavior from a client.
		 */
		r, err = protocol.ReadMessage(clientConn)
		if err == io.EOF {
			log.Info("Client rejected SSL upgrade")
			return nil
		} else if err != nil {
			return fmt.Errorf("Error reading StartupMessage after SSL upgrade: %w", err)
		} else {
			log.Info("Client accepted handshake")
		}

		version, err = r.ReadInt32()
		if err != nil {
			return fmt.Errorf("Error reading StartupMessage after SSL upgrade: %w", err)
		}
	}

	if version != protocol.ProtocolVersion {
		fmt.Printf("Invalid protocol version from client: %v", version)
		// log
		// send error to client
		// disconnect client
		return nil
	}

	hostPort, newStartupMessage, err := parseStartupMessage(r)
	if err != nil {
		fmt.Printf("Unable to parse startup message from client: %v", err)
		// log
		// send error to client
		// disconnect client
		return nil
	}

	serverConn, err := Connect(hostPort, p.SSLConfig)
	if err != nil {
		fmt.Printf("Unable to connect to backend %v: %v", hostPort, err)
		// log
		// send error to client
		// disconnect client
	}

	err = newStartupMessage.WriteTo(serverConn)
	if err != nil {
		fmt.Printf("Error writing StartupMessage to remote server: %v", err)
		// log
		// send error to client
		// disconnect client
		return nil
	}

	// start forwarding from server back to client
	serverDone := make(chan error)
	go func() {
		_, err := io.Copy(serverConn, clientConn)
		serverDone <- err
		close(serverDone)
	}()

	clientDone := make(chan error)
	go func() {
		_, err := io.Copy(clientConn, serverConn)
		clientDone <- err
		close(clientDone)
	}()

	clientErr := <-clientDone
	serverErr := <-serverDone
	serverConn.Close()
	clientConn.Close()

	log.Infof("Server: %v, Client: %v", serverErr, clientErr)

	return nil
}
