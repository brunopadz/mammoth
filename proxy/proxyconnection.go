package proxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/twooster/pg-jump/config"
	"github.com/twooster/pg-jump/protocol"
)

type ProxyConnection struct {
	log logrus.FieldLogger
	c   *config.Config
}

func parseStartupMessage(r *protocol.Reader) (hostPort string, user string, newStartupMessage *protocol.Buffer, e error) {
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
		} else if key == "user" {
			user = val
		} else {
			props[key] = val
		}
	}

	if user == "" {
		return "", "", nil, errors.New("user field empty")
	}
	if database == "" {
		return "", "", nil, errors.New("database field empty")
	}
	// parse database: host:port/database
	var split []string
	split = strings.SplitN(database, "/", 2)
	if len(split) != 2 {
		return "", "", nil, errors.New("Database string missing /")
	}
	hostPort = split[0]

	props["database"] = split[1]
	props["user"] = user

	newStartupMessage = protocol.NewBuffer()
	newStartupMessage.WriteInt32(protocol.ProtocolVersion)
	for k, v := range props {
		newStartupMessage.WriteString(k)
		newStartupMessage.WriteString(v)
	}
	newStartupMessage.WriteByte(0x00)
	return
}

func (p *ProxyConnection) HandleConnection(clientConn net.Conn) error {
	defer clientConn.Close()

	p.log.Infof("Accepting connection")

	r, err := protocol.ReadMessage(clientConn)
	if err != nil {
		p.log.Errorf("Error reading initial StartupMessage: %w", err)
		return err
	}

	version, err := r.ReadInt32()
	if err != nil {
		p.log.Errorf("Error reading initial StartupMessage: %w", err)
		return err
	}

	if version == protocol.SSLRequestCode {
		p.log.Debugf("Client requesting SSL upgrade")

		if err := r.Finalize(); err != nil {
			p.log.Errorf("Error upgrading to SSL connection: %w", err)
			return err
		}

		if p.c.Server.BaseTLSConfig != nil {
			p.log.Debugf("Upgrading SSL connection")
			_, err := clientConn.Write([]byte{protocol.SSLAllowed})
			if err != nil {
				p.log.Errorf("Error upgrading SSL connection: %w", err)
				return err
			}

			// Upgrade the connection
			sslConn := tls.Server(clientConn, p.c.Server.BaseTLSConfig.Clone())
			if err = sslConn.Handshake(); err != nil {
				p.log.Warnf("Error performing SSL handshake: %v", err)
				return err
			}
			clientConn = sslConn
		} else {
			p.log.Debugf("SSL disabled, rejecting SSL handshake")
			_, err := clientConn.Write([]byte{protocol.SSLNotAllowed})
			if err != nil {
				p.log.Errorf("Error rejecting SSL upgrade: %v", err)
				return err
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
			p.log.Info("Client rejected SSL response")
			return nil
		} else if err != nil {
			p.log.Errorf("Error reading StartupMessage after SSL response: %w", err)
			return err
		} else {
			p.log.Debugf("Client accepted SSL response")
		}

		// Re-read protocol version
		version, err = r.ReadInt32()
		if err != nil {
			p.log.Errorf("Error reading StartupMessage after SSL upgrade: %w", err)
			return err
		}
	} else { // non-SSL startup packet
		if p.c.Server.BaseTLSConfig != nil && p.c.Server.AllowUnencrypted == false {
			p.log.Info("Rejecting client without SSL because allowUnecrypted is false")
			return errors.New("Rejecting client not using SSL")
		}
	}

	if version != protocol.ProtocolVersion {
		p.log.Errorf("Invalid protocol version from client: %v", version)
		// send error to client
		return err
	}

	hostPort, user, newStartupMessage, err := parseStartupMessage(r)
	if err != nil {
		p.log.Errorf("Unable to parse startup message from client: %v", err)
		// send error to client
		return err
	}
	p.log = p.log.WithFields(logrus.Fields{
		"user":    user,
		"backend": hostPort,
	})

	p.log.Info("Connecting to backend")
	serverConn, err := p.ConnectBackend(hostPort)
	if err != nil {
		p.log.Errorf("Unable to connect to backend %v: %v", hostPort, err)
		// send error to client
		return err
	}
	defer serverConn.Close()

	err = newStartupMessage.WriteTo(serverConn)
	if err != nil {
		p.log.Errorf("Error writing StartupMessage to remote server: %v", err)
		// send error to client
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

	if clientErr != nil {
		p.log.Errorf("Client closed with error:", clientErr)
	}
	if serverErr != nil {
		p.log.Errorf("Server closed with error:", serverErr)
	}

	p.log.Infof("Client disconneted")

	return nil
}

func (p *ProxyConnection) ConnectBackend(host string) (net.Conn, error) {
	conn, err := net.Dial("tcp", host)

	if err != nil {
		return nil, err
	}

	if !p.c.Client.TrySSL {
		return conn, nil
	}

	p.log.Debug("Attempting backend SSL upgrade")
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
	err = message.WriteTo(conn)

	if err != nil {
		return nil, fmt.Errorf("Error writing to backend: %w", err)
	}

	/* Receive SSL response message. */
	sslResponseBuf := []byte{0}
	_, err = conn.Read(sslResponseBuf)

	if err != nil {
		return nil, fmt.Errorf("Error reading from backend: %w", err)
	}

	/*
	* If SSL is not allowed by the backend then close the connection and
	* throw an error.
	 */
	if sslResponseBuf[0] != protocol.SSLAllowed {
		p.log.Debug("Backend SSL unsupported")
		if !p.c.Client.AllowUnencrypted {
			conn.Close()
			return nil, errors.New("Backend does not support SSL")
		}
		p.log.Debug("Continuing with unencrypted connection")
	} else {
		p.log.Debug("Attempting to upgrade backend connection to SSL")

		sslConn := tls.Client(conn, p.c.Client.BaseTLSConfig.Clone())
		if err = sslConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("Error upgrading to SSL: %w", err)
		}

		p.log.Debug("Connection successfully upgraded")
		conn = sslConn
	}

	return conn, nil
}
