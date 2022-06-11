package config

import (
	"os"
)

type Config struct {
	RedisConfig *RedisConfig
	DBConfig    *DBConfig
}

type RedisConfig struct {
	Dsn string
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
	return &Config{
		RedisConfig: &RedisConfig{Dsn: getenv("REDIS_DSN", "localhost:6379")},
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
