package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                int
	DBPath              string
	DiscordClientID     string
	DiscordClientSecret string
	DiscordGuildID      string
	DiscordRedirectURI  string
	SessionSecret       []byte
	CookieSecure        bool // derived from DiscordRedirectURI scheme
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

	discordClientID, e1 := envRequired("DISCORD_CLIENT_ID")
	discordClientSecret, e2 := envRequired("DISCORD_CLIENT_SECRET")
	discordGuildID, e3 := envRequired("DISCORD_GUILD_ID")
	discordRedirectURI, e4 := envRequired("DISCORD_REDIRECT_URI")
	sessionSecret, e5 := envRequired("SESSION_SECRET")
	if e5 == nil && len(sessionSecret) < 32 {
		e5 = fmt.Errorf("SESSION_SECRET must be at least 32 bytes (got %d)", len(sessionSecret))
	}

	if err := errors.Join(e1, e2, e3, e4, e5); err != nil {
		return nil, err
	}

	return &Config{
		Port:                port,
		DBPath:              dbPath,
		DiscordClientID:     discordClientID,
		DiscordClientSecret: discordClientSecret,
		DiscordGuildID:      discordGuildID,
		DiscordRedirectURI:  discordRedirectURI,
		SessionSecret:       []byte(sessionSecret),
		CookieSecure:        strings.HasPrefix(discordRedirectURI, "https://"),
	}, nil
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

func envRequired(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("%s is required and not set", key)
	}
	return val, nil
}
