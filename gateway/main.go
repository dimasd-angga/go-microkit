package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimasd-angga/go-microkit/gateway/router"
	"github.com/dimasd-angga/go-microkit/internal/auth"
	"github.com/dimasd-angga/go-microkit/internal/config"
	"github.com/dimasd-angga/go-microkit/internal/grpcx"
	"github.com/dimasd-angga/go-microkit/internal/logger"
	"github.com/dimasd-angga/go-microkit/internal/tracer"
	orderpb "github.com/dimasd-angga/go-microkit/proto/order"
	userpb "github.com/dimasd-angga/go-microkit/proto/user"
	"github.com/joho/godotenv"
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
		ServiceName: "gateway",
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

	userConn, err := grpcx.Dial(fmt.Sprintf("user:%d", cfg.UserGRPCPort))
	if err != nil {
		logger.Error(ctx, err, "dial user service")
		os.Exit(1)
	}
	defer userConn.Close()

	orderConn, err := grpcx.Dial(fmt.Sprintf("order:%d", cfg.OrderGRPCPort))
	if err != nil {
		logger.Error(ctx, err, "dial order service")
		os.Exit(1)
	}
	defer orderConn.Close()

	signer := auth.NewSigner(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	h := router.New(router.Deps{
		Signer:      signer,
		UserClient:  userpb.NewUserServiceClient(userConn),
		OrderClient: orderpb.NewOrderServiceClient(orderConn),
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.GatewayPort),
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info(ctx, "gateway listening", "port", cfg.GatewayPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, err, "http serve")
		}
	}()

	<-ctx.Done()
	logger.Info(ctx, "shutting down gateway")

	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(stopCtx); err != nil {
		logger.Error(ctx, err, "http shutdown")
	}
}
