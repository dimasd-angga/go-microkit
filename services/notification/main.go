package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimasd-angga/go-microkit/internal/config"
	"github.com/dimasd-angga/go-microkit/internal/logger"
	"github.com/dimasd-angga/go-microkit/internal/messaging"
	"github.com/dimasd-angga/go-microkit/internal/tracer"
	"github.com/dimasd-angga/go-microkit/services/notification/dedupe"
	"github.com/dimasd-angga/go-microkit/services/notification/handler"
	"github.com/dimasd-angga/go-microkit/services/notification/sender"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
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

	shutdown, err := tracer.Init(ctx, tracer.Options{
		ServiceName: "notification-service",
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
		_ = shutdown(c)
	}()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error(ctx, err, "redis ping")
		os.Exit(1)
	}

	nc, err := nats.Connect(cfg.NATSURL,
		nats.Name("notification-service"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		logger.Error(ctx, err, "nats connect")
		os.Exit(1)
	}
	defer nc.Drain()

	// Make sure the stream is there. In practice the order-service owns it,
	// but this lets the notification service boot first if order isn't up yet.
	if err := messaging.EnsureStream(nc, messaging.StreamOrders,
		messaging.SubjOrderCreated,
		messaging.SubjOrderUpdated,
		messaging.SubjOrderPaid,
		messaging.SubjOrderFailed,
	); err != nil {
		logger.Error(ctx, err, "ensure stream")
		os.Exit(1)
	}

	sub, err := messaging.NewSubscriber(nc)
	if err != nil {
		logger.Error(ctx, err, "init subscriber")
		os.Exit(1)
	}

	h := &handler.Handlers{
		Sender: sender.Stdout{},
		Dedupe: dedupe.New(rdb, 24*time.Hour),
	}

	go runSub(ctx, sub, "notification-order-created", messaging.SubjOrderCreated, h.OnOrderCreated)
	go runSub(ctx, sub, "notification-order-paid", messaging.SubjOrderPaid, h.OnOrderPaid)

	logger.Info(ctx, "notification-service running")
	<-ctx.Done()
	logger.Info(ctx, "shutting down notification-service")
}

func runSub(ctx context.Context, sub *messaging.Subscriber, durable, subject string, fn messaging.Handler) {
	if err := sub.Subscribe(ctx, messaging.StreamOrders, durable, subject, fn); err != nil {
		logger.Error(ctx, err, "subscription stopped", "subject", subject)
	}
}
