package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppAddr      string
	DBHost       string
	DBPort       string
	DBName       string
	DBUser       string
	DBPassword   string
	AuthUser     string
	AuthPassword string
}

func Load() Config {
	return Config{
		AppAddr:      getEnv("APP_ADDR", ":8080"),
		DBHost:       getEnv("DB_HOST", "127.0.0.1"),
		DBPort:       getEnv("DB_PORT", "3306"),
		DBName:       getEnv("DB_NAME", "crud_db"),
		DBUser:       getEnv("DB_USER", "root"),
		DBPassword:   getEnv("DB_PASSWORD", "hastur"),
		AuthUser:     getEnv("AUTH_USER", "admin"),
		AuthPassword: getEnv("AUTH_PASSWORD", ""),
	}
}

func (c Config) DatabaseDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
