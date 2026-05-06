package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	accountapi "zxc/api/account"
	payloadapi "zxc/api/payload"
	releaseapi "zxc/api/release"
	sessionapi "zxc/api/session"
	systemapi "zxc/api/system"
	targetapi "zxc/api/target"
	tenantapi "zxc/api/tenant"
	userapi "zxc/api/user"
	"zxc/internal/config"
	"zxc/internal/infra"
	"zxc/internal/models"
	"zxc/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	configPath := flag.String("config", "config.toml", "path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded", "path", *configPath)

	creds, err := cfg.TLS.ServerCreds()
	if err != nil {
		slog.Error("Failed to load TLS credentials", "error", err)
		os.Exit(1)
	}

	database, err := infra.NewConnection(cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := infra.RunRootMigrations(database); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	if cfg.Root == uuid.Nil {
		slog.Error("root ID must be set in config")
		os.Exit(1)
	}
	var root models.User
	if err := database.Where("id = ?", cfg.Root).First(&root).Error; err != nil {
		slog.Error("Failed to load root", "error", err)
		os.Exit(1)
	}

	user := service.NewUser()
	sys := service.NewSystem()
	account := service.NewAccount()
	session := service.NewSession()
	tenant := service.NewTenant(database, cfg, &root)
	release := service.NewRelease()
	target := service.NewTarget()
	payload := service.NewPayload()

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			service.UserInterceptor(database, root.ID),
		),
	)
	userapi.RegisterUserServiceServer(grpcServer, user)
	systemapi.RegisterSystemServiceServer(grpcServer, sys)
	accountapi.RegisterAccountServiceServer(grpcServer, account)
	sessionapi.RegisterSessionServiceServer(grpcServer, session)
	tenantapi.RegisterTenantServiceServer(grpcServer, tenant)
	releaseapi.RegisterReleaseServiceServer(grpcServer, release)
	targetapi.RegisterTargetServiceServer(grpcServer, target)
	payloadapi.RegisterPayloadServiceServer(grpcServer, payload)
	reflection.Register(grpcServer)

	addr := fmt.Sprintf(":%d", 50051)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("Failed to listen", "addr", addr, "error", err)
		os.Exit(1)
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		slog.Info("Shutting down gRPC server")
		done := make(chan struct{})

		go func() {
			grpcServer.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("Server stopped gracefully")
		case <-time.After(30 * time.Second):
			slog.Info("Shutdown timeout reached, forcing stop")
			grpcServer.Stop()
		}
	}()

	slog.Info("Starting gRPC server", "addr", addr)

	if err := grpcServer.Serve(listener); err != nil {
		slog.Error("Failed to serve", "error", err)
		os.Exit(1)
	}
}
