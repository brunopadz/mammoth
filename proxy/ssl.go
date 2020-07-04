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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/twooster/pg-jump/config"
	"github.com/twooster/pg-jump/util/log"
)

/* SSL constants. */
const (
	/* SSL Modes */
	SSL_MODE_REQUIRE     string = "require"
	SSL_MODE_VERIFY_CA   string = "verify-ca"
	SSL_MODE_VERIFY_FULL string = "verify-full"
	SSL_MODE_DISABLE     string = "disable"
)

/*
 * Upgrades an incoming connection to the proxy server to use SSL
 */
func UpgradeServerConnection(conn net.Conn, ssl *config.SSLConfig) (net.Conn, error) {
	tlsConfig := tls.Config{}

	cert, err := tls.LoadX509KeyPair(
		ssl.SSLServerCert,
		ssl.SSLServerKey)

	if err != nil {
		return conn, err
	}

	tlsConfig.Certificates = []tls.Certificate{cert}

	conn = tls.Server(conn, &tlsConfig)

	return conn, nil
}

/*
 * Upgrades a connection from the proxy server to the remote Postgres
 * instance to use SSL
 */
func UpgradeClientConnection(hostPort string, conn net.Conn, ssl *config.SSLConfig) (net.Conn, error) {
	hostname, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return conn, err
	}

	verifyCA := false
	tlsConfig := tls.Config{}

	/*
	 * Configure the connection based on the mode specificed in the proxy
	 * configuration. Valid mode options are 'require', 'verify-ca',
	 * 'verify-full' and 'disable'. Any other value will result in a fatal
	 * error.
	 */
	switch ssl.SSLMode {
	case SSL_MODE_REQUIRE:
		tlsConfig.InsecureSkipVerify = true

		/*
		 * According to the documentation provided by
		 * https://www.postgresql.org/docs/current/static/libpq-ssl.html, for
		 * backwards compatibility with earlier version of PostgreSQL, if the
		 * root CA file exists, then the behavior of 'sslmode=require' needs to
		 * be the same as 'sslmode=verify-ca'.
		 */
		verifyCA = (ssl.SSLRootCA != "")
	case SSL_MODE_VERIFY_CA:
		tlsConfig.InsecureSkipVerify = true
		verifyCA = true
	case SSL_MODE_VERIFY_FULL:
		tlsConfig.ServerName = hostname
	case SSL_MODE_DISABLE:
		return conn, nil
	default:
		return conn, fmt.Errorf("Unsupported sslmode %s\n", ssl.SSLMode)
	}

	/* Add client SSL certificate and key. */
	log.Debug("Loading SSL certificate and key")
	cert, err := tls.LoadX509KeyPair(ssl.SSLCert, ssl.SSLKey)
	if err != nil {
		return conn, err
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	/* Add root CA certificate. */
	log.Debug("Loading root CA.")
	tlsConfig.RootCAs = x509.NewCertPool()
	rootCA, err := ioutil.ReadFile(ssl.SSLRootCA)
	if err != nil {
		return conn, err
	}
	tlsConfig.RootCAs.AppendCertsFromPEM(rootCA)

	/* Upgrade the connection. */
	log.Info("Upgrading to SSL connection.")
	client := tls.Client(conn, &tlsConfig)

	if verifyCA {
		log.Debug("Verify CA is enabled")
		err = verifyCertificateAuthority(client, &tlsConfig)
		return client, err
	} else {
		return client, nil
	}
}

/*
 * This function will perform a TLS handshake with the server and to verify the
 * certificates against the CA.
 *
 * client - the TLS client connection.
 * tlsConfig - the configuration associated with the connection.
 */
func verifyCertificateAuthority(client *tls.Conn, tlsConf *tls.Config) error {
	err := client.Handshake()

	if err != nil {
		return err
	}

	/* Get the peer certificates. */
	certs := client.ConnectionState().PeerCertificates

	/* Setup the verification options. */
	options := x509.VerifyOptions{
		DNSName:       client.ConnectionState().ServerName,
		Intermediates: x509.NewCertPool(),
		Roots:         tlsConf.RootCAs,
	}

	for i, certificate := range certs {
		/*
		 * The first certificate in the list is client certificate and not an
		 * intermediate certificate. Therefore it should not be added.
		 */
		if i == 0 {
			continue
		}

		options.Intermediates.AddCert(certificate)
	}

	/* Verify the client certificate.
	 *
	 * The first certificate in the certificate to verify.
	 */
	_, err = certs[0].Verify(options)

	return err
}
