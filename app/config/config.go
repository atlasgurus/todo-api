package config

import (
	"os"
	"time"
)

type Config struct {
	AuthConfig *AuthConfig
	DBConfig   *DBConfig
}

type AuthConfig struct {
	RedisDsn        string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type DBConfig struct {
	Impl     string
	Dialect  string
	Name     string
	Username string
	Password string
	Host     string
	Port     string
	SSLMode  string
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); len(value) != 0 {
		return value
	}
	return fallback
}

func GetConf() *Config {
	os.Setenv("ACCESS_SECRET", "jdnfksdmfksd")         //this should be in an env file
	os.Setenv("REFRESH_SECRET", "mcmvmkmsdnfsdmfdsjf") //this should be in an env file
	return &Config{
		AuthConfig: &AuthConfig{
			RedisDsn:        getenv("REDIS_DSN", "localhost:6379"),
			AccessTokenTTL:  15 * 60,
			RefreshTokenTTL: 60 * 24},
		DBConfig: &DBConfig{
			Impl:     getenv("TODO_DB_IMPL", "gorm"),
			Dialect:  getenv("TODO_DB_DIALECT", "postgres"),
			Name:     getenv("TODO_DB_NAME", "todo"),
			Username: getenv("TODO_DB_USERNAME", "postgres"),
			Password: getenv("TODO_DB_PASSWORD", "password"),
			Host:     getenv("TODO_DB_HOST", "localhost"),
			Port:     getenv("TODO_DB_PORT", "55000"),
			SSLMode:  getenv("TODO_DB_SSLMODE", "disable"),
		},
	}
}
