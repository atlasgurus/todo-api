package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AuthConfig *AuthConfig
	DBConfig   *DBConfig
	SMTPConfig *SMTPConfig
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
	Port     int
	SSLMode  string
}

type SMTPConfig struct {
	Host               string
	Port               int
	Username           string
	Password           string
	DoNotReplyEmail    string
	InsecureSkipVerify bool
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); len(value) != 0 {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if value := os.Getenv(key); len(value) != 0 {
		result, err := strconv.Atoi(value)
		if err == nil {
			return result
		}
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
			Port:     getenvInt("TODO_DB_PORT", 55000),
			SSLMode:  getenv("TODO_DB_SSLMODE", "disable"),
		},
		SMTPConfig: &SMTPConfig{
			Username:           getenv("TODO_SMTP_USERNAME", "test@google.com"),
			Password:           getenv("TODO_SMTP_PASSWORD", "password"),
			Host:               getenv("TODO_SMTP_HOST", "smtp.freesmtpservers.com"),
			DoNotReplyEmail:    "donotreply@gmail.com",
			Port:               getenvInt("TODO_SMTP_PORT", 25),
			InsecureSkipVerify: false,
		},
	}
}
