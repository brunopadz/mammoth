package proxy

import (
	"crypto/tls"
	"encoding/binary"
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
	log     logrus.FieldLogger
	c       *config.Config
	secrets *BackendSecrets
}

func parseStartupMessage(r *protocol.Reader) (host, port, user string, newStartupMessage *protocol.Buffer, e error) {
	var database string
	props := map[string]string{}

	for {
		key, err := r.ReadString()
		if err != nil {
			e = err
			return
		}

		// startupmessage is terminated by a 0x00 byte, which
		// will be parsed as an empty string key
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
		e = errors.New("user field empty")
		return
	}
	if database == "" {
		e = errors.New("database field empty")
		return
	}
	// parse database: host:port/database
	var split []string
	split = strings.SplitN(database, "/", 2)
	if len(split) != 2 {
		e = errors.New("Database string missing /")
		return
	}

	hostPort := split[0]

	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
		port = "5432"
	}

	props["database"] = split[1]
	props["user"] = user

	newStartupMessage = protocol.NewBuffer()
	newStartupMessage.WriteInt32(protocol.ProtocolVersion)
	for k, v := range props {
		newStartupMessage.WriteString(k)
		newStartupMessage.WriteString(v)
	}
	newStartupMessage.WriteByte(0)
	return
}

func (p *ProxyConnection) TrySSLUpgrade(conn net.Conn) (net.Conn, error) {
	if p.c.Server.BaseTLSConfig != nil {
		p.log.Debug("Upgrading SSL connection")
		_, err := conn.Write([]byte{protocol.SSLAllowed})
		if err != nil {
			return conn, err
		}

		// Upgrade the connection
		sslConn := tls.Server(conn, p.c.Server.BaseTLSConfig.Clone())
		return sslConn, sslConn.Handshake()
	} else {
		p.log.Debug("SSL disabled, rejecting SSL handshake")
		_, err := conn.Write([]byte{protocol.SSLNotAllowed})
		return conn, err
	}
}

func (p *ProxyConnection) HandleConnection(clientConn net.Conn) error {
	defer clientConn.Close()

	r, err := protocol.ReadMessage(clientConn)
	if err != nil {
		p.log.Infof("Error reading initial StartupMessage: %w", err)
		return err
	}

	version, err := r.ReadInt32()
	if err != nil {
		p.log.Infof("Error reading initial StartupMessage: %w", err)
		return err
	}

	if version == protocol.SSLRequestCode {
		p.log.Debugf("Client requesting SSL upgrade")

		if err := r.Finalize(); err != nil {
			p.log.Infof("Malformed SSLRequest packet: %w", err)
			return err
		}

		clientConn, err = p.TrySSLUpgrade(clientConn)
		if err != nil {
			p.log.Infof("Error performing SSL handshake: %w")
			return err
		}
		/*
		 * Re-read the startup message from the client. It is possible that the
		 * client might not like the response given and as a result it might
		 * close the connection. This is not an 'error' condition as this is an
		 * expected behavior from a client.
		 */
		r, err = protocol.ReadMessage(clientConn)
		if err == io.EOF {
			p.log.Info("Client rejected SSL response and closed connection")
			return nil
		} else if err != nil {
			p.log.Infof("Error reading StartupMessage after SSL handshake: %w", err)
			return err
		}

		// Re-read protocol version
		version, err = r.ReadInt32()
		if err != nil {
			p.log.Infof("Error reading StartupMessage after SSL handshake: %w", err)
			return err
		}
	} else if version != protocol.CancelRequestCode {
		// NB: psql does not attempt an SSL upgrade when accepting cancel packets.
		// For this reason, we accept cancel requests even if it's not SSL
		if p.c.Server.BaseTLSConfig != nil && p.c.Server.AllowUnencrypted == false {
			p.log.Infof("Rejecting client without SSL because allowUnecrypted is false (version = %v)", version)
			return nil
		}
	}

	if version == protocol.CancelRequestCode {
		pid, err := r.ReadInt32()
		if err != nil {
			return err
		}
		secret, err := r.ReadInt32()
		if err != nil {
			return err
		}
		if err := r.Finalize(); err != nil {
			return err
		}
		s, ok := p.secrets.Get(pid, secret)
		if !ok {
			return nil
		}

		p.log.Debug("Connecting to backend for cancellation")
		serverConn, err := p.ConnectBackend(s.host, s.port)
		if err != nil {
			p.log.Infof("Unable to connect to backend for cancellation %v:%v: %v", s.host, s.port, err)
		}
		defer serverConn.Close()

		msg := protocol.NewBuffer()
		msg.WriteInt32(protocol.CancelRequestCode)
		msg.WriteInt32(pid)
		msg.WriteInt32(s.origSecret)
		if err := msg.WriteTo(serverConn); err != nil {
			return err
		}

		p.log.Infof("Successfully sent cancellation to %v:%v, pid %v", s.host, s.port, pid)
		// In theory, the server should drop the connection immediately after, so
		// we don't await a response, and neither should the client.
		return nil
	} else if version != protocol.ProtocolVersion {
		p.log.Infof("Unsupported protocol version from client: %v", version)
		return nil
	}

	host, port, user, newStartupMessage, err := parseStartupMessage(r)
	if err != nil {
		p.log.Infof("Unable to parse startup message from client: %v", err)
		protocol.WriteError(clientConn, protocol.Error{
			Severity: protocol.ErrorSeverityFatal,
			Code:     protocol.ErrorCodeConnectionFailure,
			Message:  "Unable to parse connection string",
			Detail:   err.Error(),
		})
		return err
	}

	p.log = p.log.WithFields(logrus.Fields{
		"user":   user,
		"server": net.JoinHostPort(host, port),
	})

	if p.c.HostRegex != nil && !p.c.HostRegex.MatchString(host) {
		p.log.Infof("Backend host %v does not match regexp %v", host)
		protocol.WriteError(clientConn, protocol.Error{
			Severity: protocol.ErrorSeverityFatal,
			Code:     protocol.ErrorCodeConnectionFailure,
			Message:  "Remote host does not match regexp",
		})
		return nil
	}

	p.log.Debug("Connecting to backend")
	serverConn, err := p.ConnectBackend(host, port)
	if err != nil {
		p.log.Infof("Unable to connect to backend %v:%v: %v", host, port, err)
		protocol.WriteError(clientConn, protocol.Error{
			Severity: protocol.ErrorSeverityFatal,
			Code:     protocol.ErrorCodeClientUnableToConnect,
			Message:  "Unable to connect to remote backend",
			Detail:   err.Error(),
		})
		return err
	}
	defer serverConn.Close()

	err = newStartupMessage.WriteTo(serverConn)
	if err != nil {
		p.log.Errorf("Error writing StartupMessage to remote server: %v", err)
		protocol.WriteError(clientConn, protocol.Error{
			Severity: protocol.ErrorSeverityFatal,
			Code:     protocol.ErrorCodeClientUnableToConnect,
			Message:  "Network error communicating with backend server",
			Detail:   err.Error(),
		})
		return err
	}

	p.log.Debug("Passing through data between client and server")
	clientDone := make(chan bool)
	go func() {
		err := p.PassthruAndLog(serverConn, clientConn)
		if err != nil && err != io.EOF {
			p.log.Infof("Client closed with error: %v", err)
		}
		close(clientDone)
	}()

	pid, secret, added, err := p.PassthruAndRewriteBackendData(clientConn, serverConn, host, port)
	if added {
		defer p.secrets.Remove(pid, secret)
	}
	if err != nil {
		return err
	}

	// start pass-thru copy
	serverDone := make(chan bool)
	go func() {
		_, err := io.Copy(clientConn, serverConn)
		if err != nil {
			p.log.Infof("Server closed with error: %v", err)
		}
		close(serverDone)
	}()

	// TODO[tmw]: Can these hang?
	<-clientDone
	<-serverDone

	p.log.Infof("Client disconnected")

	return nil
}

// Copies data from the serverConn to the clientConn, parsing the packets
// looking for a BackendKeyData message. This will be rewritten and stored
// into the server-global secrets store and potentially given a new secret
// that will be written to the client. In this way, we can handle cancellation.
// Stops copying data after the first ReadyForQuery message is received,
// which indicates that no further BackendDataPacket will be forthcoming.
func (p *ProxyConnection) PassthruAndRewriteBackendData(clientConn, serverConn net.Conn, host, port string) (pid int32, secret int32, added bool, err error) {
	msgTypeBuf := make([]byte, 1)

	for {
		_, err = serverConn.Read(msgTypeBuf)
		if err != nil {
			return
		}
		_, err = clientConn.Write(msgTypeBuf)
		if err != nil {
			return
		}

		msgType := msgTypeBuf[0]

		var msg *protocol.Reader
		msg, err = protocol.ReadMessage(serverConn)
		if err != nil {
			return
		}

		if msgType == protocol.BackendKeyDataMessageType {
			pid, err = msg.ReadInt32()
			if err != nil {
				return
			}
			secret, err = msg.ReadInt32()
			if err != nil {
				return
			}
			// If this is the second time we've seen the packet, let's
			// remove the old entry since it's no longer relevant
			if added {
				p.secrets.Remove(pid, secret)
			}
			secret = p.secrets.Add(pid, secret, host, port)
			added = true

			msgOut := protocol.NewBuffer()
			msgOut.WriteInt32(pid)
			msgOut.WriteInt32(secret)
			err = msgOut.WriteTo(clientConn)

			if err != nil {
				return
			}
		} else {
			err = binary.Write(clientConn, binary.BigEndian, msg.Len)
			if err != nil {
				return
			}
			_, err = io.Copy(clientConn, msg)
			if err != nil {
				return
			}
			// We can stop listening once we've passed-thru the first ReadyForQuery
			if msgType == protocol.ReadyForQueryMessageType {
				return
			}
		}
	}
}

// Parses all packets coming from the client conn to the server conn,
// and logs all relevant commands to the logger
func (p *ProxyConnection) PassthruAndLog(serverConn, clientConn net.Conn) error {
	msgTypeBuf := make([]byte, 1)

	// Every byte that's read out of the tee will be sent straight to
	// the postgres server
	tee := io.TeeReader(clientConn, serverConn)
	for {
		_, err := tee.Read(msgTypeBuf)
		if err != nil {
			return err
		}
		msgType := msgTypeBuf[0]

		msg, err := protocol.ReadMessage(tee)
		if err != nil {
			return err
		}

		fields := logrus.Fields{}

		switch msgType {
		case protocol.BindMessageType:
			err = handleBind(msg, fields)

		case protocol.CloseMessageType:
			err = handleClose(msg, fields)

		case protocol.CopyDataMessageType:
			err = handleCopyData(msg, fields)

		case protocol.CopyDoneMessageType:
			err = handleCopyDone(msg, fields)

		case protocol.CopyFailMessageType:
			err = handleCopyFail(msg, fields)

		case protocol.DescribeMessageType:
			err = handleDescribe(msg, fields)

		case protocol.ExecuteMessageType:
			err = handleExecute(msg, fields)

		case protocol.FunctionCallMessageType:
			err = handleFunctionCall(msg, fields)

		case protocol.ParseMessageType:
			err = handleParse(msg, fields)

		case protocol.SimpleQueryMessageType:
			err = handleSimpleQuery(msg, fields)

		case protocol.SyncMessageType:
			err = handleSync(msg, fields)

		case protocol.TerminateMessageType:
			err = handleTerminate(msg, fields)

		default:
			fields["type"] = "Unknown"
			fields["code"] = int(msgType)
			fields["len"] = msg.Len
			err = msg.Discard()
		}

		if err != nil && err != io.EOF {
			fields["ioerror"] = err.Error()
		}

		p.log.WithFields(fields).Info("Command")
		if err != nil {
			return err
		}
	}
}

func (p *ProxyConnection) ConnectBackend(host, port string) (net.Conn, error) {
	hostPort := net.JoinHostPort(host, port)
	conn, err := net.Dial("tcp", hostPort)

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

		tlsConfig := p.c.Client.BaseTLSConfig.Clone()
		tlsConfig.ServerName = host
		sslConn := tls.Client(conn, tlsConfig)
		if err = sslConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("Error upgrading to SSL: %w", err)
		}

		p.log.Debug("Connection successfully upgraded")
		conn = sslConn
	}

	return conn, nil
}
