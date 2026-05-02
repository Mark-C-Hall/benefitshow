package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	Port int
}

func Load() (*Config, error) {
	port, err := envInt("PORT", 8080)
	if err != nil {
		return nil, err
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("PORT %d out of range", port)
	}

	return &Config{Port: port}, nil
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
