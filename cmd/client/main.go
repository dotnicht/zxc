package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"zxc/api/account"
	"zxc/api/payload"
	"zxc/api/release"
	"zxc/api/target"
	"zxc/api/tenant"
	"zxc/api/user"
	"zxc/internal/config"
)

type clientConfig struct {
	Address string     `mapstructure:"address"`
	UserID  string     `mapstructure:"userid"`
	Log     bool       `mapstructure:"log"`
	Timeout string     `mapstructure:"timeout"`
	TLS     config.TLS `mapstructure:"tls"`
}

type state struct {
	cfg     *clientConfig
	conn    *grpc.ClientConn
	tenant  tenant.TenantServiceClient
	user    user.UserServiceClient
	account account.AccountServiceClient
	target  target.TargetServiceClient
	payload payload.PayloadServiceClient
	release release.ReleaseServiceClient
}

var st state

var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "gRPC API client",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if cfg.Log {
			slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})))
			slog.Info("client starting", "userid", cfg.UserID, "address", cfg.Address)
		}
		creds, err := cfg.TLS.ClientCreds()
		if err != nil {
			return fmt.Errorf("TLS: %w", err)
		}
		conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(creds))
		if err != nil {
			return fmt.Errorf("connecting to %s: %w", cfg.Address, err)
		}
		st = state{
			cfg:     cfg,
			conn:    conn,
			tenant:  tenant.NewTenantServiceClient(conn),
			user:    user.NewUserServiceClient(conn),
			account: account.NewAccountServiceClient(conn),
			target:  target.NewTargetServiceClient(conn),
			payload: payload.NewPayloadServiceClient(conn),
			release: release.NewReleaseServiceClient(conn),
		}
		return nil
	},
}

func loadConfig() (*clientConfig, error) {
	cfg := &clientConfig{
		Address: "localhost:50051",
	}

	configDir, _ := os.UserConfigDir()
	candidates := []string{
		filepath.Join(configDir, "zxc", "client.toml"),
		"/etc/zxc/client.toml",
	}

	for _, path := range candidates {
		v := viper.New()
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err == nil {
			if err := v.Unmarshal(cfg); err != nil {
				return nil, fmt.Errorf("parsing %s: %w", path, err)
			}
			if cfg.UserID == "" {
				cfg.UserID = v.GetString("userid")
			}
			if cfg.UserID == "" {
				cfg.UserID = v.GetString("user_id")
			}
			break
		}
	}

	if cfg.Address == "" {
		return nil, fmt.Errorf("address must not be empty")
	}
	if cfg.Timeout != "" {
		if _, err := time.ParseDuration(cfg.Timeout); err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", cfg.Timeout, err)
		}
	}
	return cfg, nil
}

func main() {
	rootCmd.AddCommand(tenantCmd())
	rootCmd.AddCommand(userCmd())
	rootCmd.AddCommand(accountCmd())
	rootCmd.AddCommand(targetCmd())
	rootCmd.AddCommand(payloadCmd())
	rootCmd.AddCommand(releaseCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
