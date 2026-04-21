package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"google.golang.org/grpc/credentials"
)

type TLS struct {
	CA   string `mapstructure:"ca"`
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type Worker struct {
	TenantIDs        []string `mapstructure:"tenant_ids"`
	ExcludeTenantIDs []string `mapstructure:"exclude_tenant_ids"`
}

type Config struct {
	Database string `mapstructure:"database"`
	Storage  string `mapstructure:"storage"`
	Webhook  string `mapstructure:"webhook"`
	Root     string `mapstructure:"root"`
	TLS      TLS    `mapstructure:"tls"`
	Worker   Worker `mapstructure:"worker"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Database == "" {
		return nil, fmt.Errorf("database connection string is required")
	}
	config.Worker.normalize()

	return &config, nil
}

func (w *Worker) normalize() {
	w.TenantIDs = normalizeList(w.TenantIDs)
	w.ExcludeTenantIDs = normalizeList(w.ExcludeTenantIDs)
}

func normalizeList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
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
