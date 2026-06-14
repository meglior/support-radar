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
	return &Config{\r
		ServerAddr:      getEnv("SERVER_ADDR", ":8443"),\r
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://radar:radar@localhost:5432/support_radar?sslmode=disable"),\r
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),\r
		JWTSecret:       getEnv("JWT_SECRET", "mvp-secret-change-in-production"),\r
		TLSCertPath:     getEnv("TLS_CERT_PATH", "certs/server.crt"),\r
		TLSKeyPath:      getEnv("TLS_KEY_PATH", "certs/server.key"),\r
		RateLimitMax:    getEnvInt("RATE_LIMIT_MAX", 3),\r
		RateLimitWindow: getEnvDuration("RATE_LIMIT_WINDOW", 5*time.Minute),\r
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