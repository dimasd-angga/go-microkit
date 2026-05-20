package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimasd-angga/go-microkit/internal/auth"
	"github.com/dimasd-angga/go-microkit/internal/config"
	"github.com/dimasd-angga/go-microkit/internal/grpcx"
	"github.com/dimasd-angga/go-microkit/internal/logger"
	"github.com/dimasd-angga/go-microkit/internal/tracer"
	userpb "github.com/dimasd-angga/go-microkit/proto/user"
	"github.com/dimasd-angga/go-microkit/services/user/handler"
	"github.com/dimasd-angga/go-microkit/services/user/repository"
	"github.com/dimasd-angga/go-microkit/services/user/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // .env is optional

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger.Init(cfg.LogLevel, cfg.Env)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	shutdownTracer, err := tracer.Init(ctx, tracer.Options{
		ServiceName: "user-service",
		Endpoint:    cfg.OTELEndpoint,
		Enabled:     cfg.OTELEnabled,
		Env:         cfg.Env,
	})
	if err != nil {
		logger.Error(ctx, err, "init tracer")
		os.Exit(1)
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracer(c)
	}()

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		logger.Error(ctx, err, "connect postgres")
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error(ctx, err, "ping postgres")
		os.Exit(1)
	}

	signer := auth.NewSigner(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	repo := repository.NewPostgres(pool)
	uc := usecase.New(repo, signer)

	srv := grpcx.NewServer()
	userpb.RegisterUserServiceServer(srv, handler.NewGRPC(uc))

	lis, err := grpcx.Listen(cfg.UserGRPCPort)
	if err != nil {
		logger.Error(ctx, err, "listen")
		os.Exit(1)
	}

	go func() {
		logger.Info(ctx, "user-service grpc listening", "port", cfg.UserGRPCPort)
		if err := srv.Serve(lis); err != nil {
			logger.Error(ctx, err, "grpc serve")
		}
	}()

	<-ctx.Done()
	logger.Info(ctx, "shutting down user-service")

	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		srv.Stop()
	}
}
