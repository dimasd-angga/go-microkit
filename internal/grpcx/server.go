// Package grpcx contains small server/dial helpers shared by all gRPC servers
// in the kit. Centralising this stops every service from re-implementing the
// same interceptor stack.
package grpcx

import (
	"context"
	"net"
	"time"

	"github.com/dimasd-angga/go-microkit/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// NewServer returns a *grpc.Server with logging + recovery interceptors.
func NewServer() *grpc.Server {
	return grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              30 * time.Second,
			Timeout:           5 * time.Second,
		}),
		grpc.ChainUnaryInterceptor(
			recoveryUnary,
			loggingUnary,
		),
	)
}

// Dial is a convenience around grpc.NewClient. Plain text — wrap with mTLS
// when deploying outside a trusted network.
func Dial(target string) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(8*1024*1024),
		),
	)
}

// Listen binds a TCP listener on :port.
func Listen(port int) (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort("0.0.0.0", itoa(port)))
}

func itoa(n int) string {
	// tiny helper to avoid importing strconv just for this
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 8)
	for n > 0 {
		buf = append([]byte{digits[n%10]}, buf...)
		n /= 10
	}
	return string(buf)
}

func recoveryUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(ctx, nil, "panic in grpc handler", "method", info.FullMethod, "panic", r)
			err = grpcInternal("internal server error")
		}
	}()
	return h(ctx, req)
}

func loggingUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := h(ctx, req)
	logger.Debug(ctx, "grpc call",
		"method", info.FullMethod,
		"duration_ms", time.Since(start).Milliseconds(),
		"err", err,
	)
	return resp, err
}
