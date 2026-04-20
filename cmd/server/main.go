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

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	payloadapi "zxc/api/payload"
	releaseapi "zxc/api/release"
	targetapi "zxc/api/target"
	tenantapi "zxc/api/tenant"
	userapi "zxc/api/user"
	"zxc/internal/config"
	"zxc/internal/db"
	"zxc/internal/middleware"
	"zxc/internal/models"
	"zxc/internal/service"
	"zxc/internal/workflow"
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

	database, err := db.NewConnection(cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := db.RunRootMigrations(database); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	if cfg.Root == "" {
		slog.Error("root ID must be set in config")
		os.Exit(1)
	}
	var root models.User
	if err := database.Where("id = ?", cfg.Root).First(&root).Error; err != nil {
		slog.Error("Failed to load root", "error", err)
		os.Exit(1)
	}

	cache := db.NewCache()
	store := workflow.NewStore(database)
	user := service.NewUser(database, cache)
	tenant := service.NewTenant(database, cfg, &root)
	release := service.NewRelease(database, cache, store)
	target := service.NewTarget(database, cache, store)
	payload := service.NewPayload(database, cache)

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			middleware.User(cache, database, root.ID),
		),
	)
	userapi.RegisterUserServiceServer(grpcServer, user)
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
