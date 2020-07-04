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
	"net"

	"github.com/twooster/pg-jump/config"
	"github.com/twooster/pg-jump/protocol"
	"github.com/twooster/pg-jump/util/log"
)

func Connect(host string, ssl *config.SSLConfig) (net.Conn, error) {
	conn, err := net.Dial("tcp", host)

	if err != nil {
		return nil, err
	}

	if ssl.Enable {
		log.Info("SSL connections are enabled.")

		/*
		 * First determine if SSL is allowed by the backend. To do this, send an
		 * SSL request. The response from the backend will be a single byte
		 * message. If the value is 'S', then SSL connections are allowed and an
		 * upgrade to the connection should be attempted. If the value is 'N',
		 * then the backend does not support SSL connections.
		 */

		/* Create the SSL request message. */
		message := protocol.NewBuffer()
		message.WriteInt32(protocol.SSLRequestCode)
		err := message.WriteTo(conn)

		if err != nil {
			log.Error("Error sending SSL request to backend.")
			log.Errorf("Error: %s", err.Error())
			return nil, err
		}

		/* Receive SSL response message. */
		sslResponseBuf := []byte{0}
		_, err = conn.Read(sslResponseBuf)

		if err != nil {
			log.Error("Error receiving SSL response from backend.")
			log.Errorf("Error: %v", err)
			return nil, err
		}

		/*
		 * If SSL is not allowed by the backend then close the connection and
		 * throw an error.
		 */
		if sslResponseBuf[0] != protocol.SSLAllowed {
			log.Error("The backend does not allow SSL connections.")
			conn.Close()
		} else {
			log.Debug("SSL connections are allowed by PostgreSQL.")
			log.Debug("Attempting to upgrade connection.")
			conn, err = UpgradeClientConnection(host, conn, ssl)
			if err != nil {
				log.Debug("Connection failed to upgrade.")
				return conn, err
			}
			log.Debug("Connection successfully upgraded.")
		}
	}

	return conn, nil
}
