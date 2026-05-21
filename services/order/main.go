package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimasd-angga/go-microkit/internal/config"
	"github.com/dimasd-angga/go-microkit/internal/grpcx"
	"github.com/dimasd-angga/go-microkit/internal/logger"
	"github.com/dimasd-angga/go-microkit/internal/messaging"
	"github.com/dimasd-angga/go-microkit/internal/tracer"
	orderpb "github.com/dimasd-angga/go-microkit/proto/order"
	"github.com/dimasd-angga/go-microkit/services/order/handler"
	"github.com/dimasd-angga/go-microkit/services/order/repository"
	"github.com/dimasd-angga/go-microkit/services/order/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger.Init(cfg.LogLevel, cfg.Env)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	shutdownTracer, err := tracer.Init(ctx, tracer.Options{
		ServiceName: "order-service",
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

	nc, err := nats.Connect(cfg.NATSURL,
		nats.Name("order-service"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		logger.Error(ctx, err, "nats connect")
		os.Exit(1)
	}
	defer nc.Drain()

	if err := messaging.EnsureStream(nc, messaging.StreamOrders,
		messaging.SubjOrderCreated,
		messaging.SubjOrderUpdated,
		messaging.SubjOrderPaid,
		messaging.SubjOrderFailed,
	); err != nil {
		logger.Error(ctx, err, "ensure stream")
		os.Exit(1)
	}

	pub, err := messaging.NewPublisher(nc)
	if err != nil {
		logger.Error(ctx, err, "init publisher")
		os.Exit(1)
	}

	repo := repository.NewPostgres(pool)
	uc := usecase.New(repo, pub)

	srv := grpcx.NewServer()
	orderpb.RegisterOrderServiceServer(srv, handler.NewGRPC(uc))

	lis, err := grpcx.Listen(cfg.OrderGRPCPort)
	if err != nil {
		logger.Error(ctx, err, "listen")
		os.Exit(1)
	}

	go func() {
		logger.Info(ctx, "order-service grpc listening", "port", cfg.OrderGRPCPort)
		if err := srv.Serve(lis); err != nil {
			logger.Error(ctx, err, "grpc serve")
		}
	}()

	<-ctx.Done()
	logger.Info(ctx, "shutting down order-service")

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
