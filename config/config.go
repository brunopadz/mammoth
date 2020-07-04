package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/twooster/pg-jump/config/file"
)

type ClientTLSConfig struct {
	AllowUnencrypted bool
	TrySSL           bool
	BaseTLSConfig    tls.Config
}

type ServerTLSConfig struct {
	AllowUnencrypted bool
	BaseTLSConfig    *tls.Config
}

type Config struct {
	HostPort  string
	ClientTLS ClientTLSConfig
	ServerTLS ServerTLSConfig
}

func FromFile(f *file.Config) (*Config, error) {
	c := Config{
		HostPort: f.HostPort,
		ClientTLS: ClientTLSConfig{
			AllowUnencrypted: f.Client.AllowUnencrypted,
			TrySSL:           f.Client.TrySSL,
		},
		ServerTLS: ServerTLSConfig{
			AllowUnencrypted: f.Server.AllowUnencrypted,
		},
	}

	if f.Server.Cert != "" || f.Server.Key != "" {
		cert, err := tls.LoadX509KeyPair(f.Server.Cert, f.Server.Key)
		if err != nil {
			return nil, fmt.Errorf("Error loading server SSL keypair: %w", err)
		}
		c.ServerTLS.BaseTLSConfig.Certificates = []tls.Certificate{cert}
	} else if f.Server.AllowUnencrypted == false {
		return nil, fmt.Errorf("Server allowUnencrypted is false, but no SSL keypair specified")
	}

	if f.Client.Cert != "" || f.Client.Key != "" {
		cert, err := tls.LoadX509KeyPair(f.Client.Cert, f.Client.Key)
		if err != nil {
			return nil, fmt.Errorf("Error loading client SSL keypair: %w", err)
		}
		c.ClientTLS.BaseTLSConfig.Certificates = []tls.Certificate{cert}
	}

	if f.Client.RootCA != "" {
		rootCA, err := ioutil.ReadFile(f.Client.RootCA)
		if err != nil {
			return nil, fmt.Errorf("Error loading client Root CA: %w", err)
		}
		c.ClientTLS.BaseTLSConfig.RootCAs = x509.NewCertPool()
		c.ClientTLS.BaseTLSConfig.RootCAs.AppendCertsFromPEM(rootCA)
	}

	return &c, nil
}
