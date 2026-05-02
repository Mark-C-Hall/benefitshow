package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	Port   int
	DBPath string
}

func Load() (*Config, error) {
	port, err := envInt("PORT", 8080)
	if err != nil {
		return nil, err
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("PORT %d out of range", port)
	}

	dbPath := envWithDefault("DB_PATH", "./benefitshow.db")

	return &Config{Port: port, DBPath: dbPath}, nil
}

func (c Config) ListenAddr() string {
	return net.JoinHostPort("", strconv.Itoa(c.Port))
}

func envInt(key string, fallback int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a valid int: %w", key, val, err)
	}
	return n, nil
}

func envWithDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
