package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	RedisURL       string
	JWTSecret      string
	EncryptionKey  string
	CacheThreshold float64
	CORSOrigins    string
	SecureCookies  bool
}

func Load() *Config {
	threshold := 0.95
	if v := os.Getenv("THRIFT_CACHE_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = f
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := getEnv("THRIFT_JWT_SECRET", "dev-jwt-secret-change-me")
	if jwtSecret == "dev-jwt-secret-change-me" {
		log.Println("WARNING: Using default JWT secret. Set THRIFT_JWT_SECRET for production!")
	}

	return &Config{
		Port:           port,
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://thrift:thriftllm_secret_password@localhost:5432/thriftllm?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:      jwtSecret,
		EncryptionKey:  getEnv("THRIFT_ENCRYPTION_KEY", "dev-encryption-key-change"),
		CacheThreshold: threshold,
		CORSOrigins:    os.Getenv("THRIFT_CORS_ORIGINS"),
		SecureCookies:  os.Getenv("THRIFT_SECURE_COOKIES") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
