package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
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
	Bind      string
	ClientTLS ClientTLSConfig
	ServerTLS ServerTLSConfig
}

func FromFile(f *file.Config) (*Config, error) {
	c := Config{
		Bind: f.Bind,
		ClientTLS: ClientTLSConfig{
			AllowUnencrypted: f.Client.AllowUnencrypted,
			TrySSL:           f.Client.TrySSL,
		},
		ServerTLS: ServerTLSConfig{
			AllowUnencrypted: f.Server.AllowUnencrypted,
		},
	}

	if f.Server.Cert != "" || f.Server.Key != "" || f.Server.CA != "" {
		if f.Server.Cert == "" || f.Server.Key == "" {
			return nil, errors.New("Missing server key or cert")
		}
		cert, err := tls.LoadX509KeyPair(f.Server.Cert, f.Server.Key)
		if err != nil {
			return nil, fmt.Errorf("Error loading server SSL keypair: %w", err)
		}
		c.ServerTLS.BaseTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if f.Server.CA != "" {
			clientCA, err := ioutil.ReadFile(f.Server.CA)
			if err != nil {
				return nil, fmt.Errorf("Error loading server Client CA: %w", err)
			}
			c.ServerTLS.BaseTLSConfig.ClientCAs = x509.NewCertPool()
			c.ServerTLS.BaseTLSConfig.ClientCAs.AppendCertsFromPEM(clientCA)
		}
	} else if f.Server.AllowUnencrypted == false {
		return nil, fmt.Errorf("Server allowUnencrypted is false, but no SSL keypair specified")
	}

	if f.Client.Cert != "" || f.Client.Key != "" {
		if f.Client.Cert == "" || f.Client.Key == "" {
			return nil, errors.New("Missing client key or cert")
		}
		cert, err := tls.LoadX509KeyPair(f.Client.Cert, f.Client.Key)
		if err != nil {
			return nil, fmt.Errorf("Error loading client SSL keypair: %w", err)
		}
		c.ClientTLS.BaseTLSConfig.Certificates = []tls.Certificate{cert}
	}

	if f.Client.CA != "" {
		rootCA, err := ioutil.ReadFile(f.Client.CA)
		if err != nil {
			return nil, fmt.Errorf("Error loading client Root CA: %w", err)
		}
		c.ClientTLS.BaseTLSConfig.RootCAs = x509.NewCertPool()
		c.ClientTLS.BaseTLSConfig.RootCAs.AppendCertsFromPEM(rootCA)
	}

	return &c, nil
}
