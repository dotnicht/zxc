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
	"zxc/api/payload"
	"zxc/api/release"
	"zxc/api/target"
	"zxc/api/tenant"
	"zxc/api/user"
	"zxc/internal/config"
	"zxc/internal/db"
	"zxc/internal/logger"
	"zxc/internal/middleware"
	"zxc/internal/models"
	"zxc/internal/service"
)

func main() {
	logger.Init()

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
	userSvc := service.NewUser(database, cache)
	tenantSvc := service.NewTenant(database, cfg, &root)
	releaseSvc := service.NewRelease(database, cache)
	targetSvc := service.NewTargetSvc(database, cache)
	payloadSvc := service.NewPayload(database, cache)

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			middleware.User(cache, database),
		),
	)
	user.RegisterUserServiceServer(grpcServer, userSvc)
	tenant.RegisterTenantServiceServer(grpcServer, tenantSvc)
	release.RegisterReleaseServiceServer(grpcServer, releaseSvc)
	target.RegisterTargetServiceServer(grpcServer, targetSvc)
	payload.RegisterPayloadServiceServer(grpcServer, payloadSvc)
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
