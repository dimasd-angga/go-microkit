// Package config loads runtime config from environment variables.
//
// We deliberately avoid YAML/Viper. Every value comes from env, validated on
// startup. If something required is missing, we crash with a clear message
// instead of running with silent defaults.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string
	LogLevel string

	UserGRPCPort         int
	OrderGRPCPort        int
	NotificationGRPCPort int
	GatewayPort          int

	PostgresDSN string

	RedisAddr     string
	RedisPassword string

	NATSURL string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	OTELEndpoint    string
	OTELServiceName string
	OTELEnabled     bool
}

// Load reads config from the process environment. .env files should already
// be loaded by godotenv at the caller.
func Load() (*Config, error) {
	c := &Config{
		Env:                  getStr("ENV", "development"),
		LogLevel:             getStr("LOG_LEVEL", "info"),
		UserGRPCPort:         getInt("USER_SERVICE_GRPC_PORT", 50051),
		OrderGRPCPort:        getInt("ORDER_SERVICE_GRPC_PORT", 50052),
		NotificationGRPCPort: getInt("NOTIFICATION_SERVICE_GRPC_PORT", 50053),
		GatewayPort:          getInt("GATEWAY_PORT", 8080),
		PostgresDSN:          os.Getenv("POSTGRES_DSN"),
		RedisAddr:            getStr("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        os.Getenv("REDIS_PASSWORD"),
		NATSURL:              getStr("NATS_URL", "nats://localhost:4222"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		JWTAccessTTL:         getDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:        getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		OTELEndpoint:         getStr("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		OTELServiceName:      getStr("OTEL_SERVICE_NAME", "go-microkit"),
		OTELEnabled:          getBool("OTEL_ENABLED", true),
	}

	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.PostgresDSN == "" {
		missing = append(missing, "POSTGRES_DSN")
	}
	if c.JWTSecret == "" || c.JWTSecret == "change-me-in-production-please" {
		if c.Env != "development" {
			missing = append(missing, "JWT_SECRET (must be set outside of development)")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (c *Config) IsProd() bool { return c.Env == "production" }

func getStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getBool(key string, def bool) bool {
	v := strings.ToLower(os.Getenv(key))
	switch v {
	case "":
		return def
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func getDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
