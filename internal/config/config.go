package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerAddr      string
	DatabaseURL     string
	RedisAddr       string
	JWTSecret       string
	TLSCertPath     string
	TLSKeyPath      string
	RateLimitMax    int
	RateLimitWindow time.Duration
}

func Load() *Config {
	return &Config{
		ServerAddr:      getEnv("SERVER_ADDR", ":8443"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://radar:radar@localhost:5432/support_radar?sslmode=disable"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:       getEnv("JWT_SECRET", "mvp-secret-change-in-production"),
		TLSCertPath:     getEnv("TLS_CERT_PATH", "certs/server.crt"),
		TLSKeyPath:      getEnv("TLS_KEY_PATH", "certs/server.key"),
		RateLimitMax:    getEnvInt("RATE_LIMIT_MAX", 3),
		RateLimitWindow: getEnvDuration("RATE_LIMIT_WINDOW", 5*time.Minute),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if val, err := time.ParseDuration(v); err == nil {
			return val
		}
	}
	return defaultVal
}
