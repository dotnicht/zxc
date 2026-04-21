package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"google.golang.org/grpc/credentials"
)

type TLS struct {
	CA   string
	Cert string
	Key  string
}

type Worker struct {
	ID uuid.UUID
}

type Config struct {
	Database string
	Storage  string
	Webhook  string
	Root     uuid.UUID
	TLS      TLS
	Worker   Worker
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.TextUnmarshallerHookFunc(),
		),
	)); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Database == "" {
		return nil, fmt.Errorf("database connection string is required")
	}
	return &config, nil
}

func (t TLS) ServerCreds() (credentials.TransportCredentials, error) {
	if t.CA == "" || t.Cert == "" || t.Key == "" {
		return nil, fmt.Errorf("tls.ca, tls.cert, and tls.key are required")
	}

	cert, err := tls.LoadX509KeyPair(t.Cert, t.Key)
	if err != nil {
		return nil, fmt.Errorf("load server keypair: %w", err)
	}

	ca, err := os.ReadFile(t.CA)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("parse CA cert")
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}), nil
}

func (t TLS) ClientCreds() (credentials.TransportCredentials, error) {
	if t.CA == "" || t.Cert == "" || t.Key == "" {
		return nil, fmt.Errorf("tls.ca, tls.cert, and tls.key are required")
	}

	cert, err := tls.LoadX509KeyPair(t.Cert, t.Key)
	if err != nil {
		return nil, fmt.Errorf("load client keypair: %w", err)
	}

	ca, err := os.ReadFile(t.CA)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("parse CA cert")
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}), nil
}
