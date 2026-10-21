package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment  string
	HTTPAddress  string
	DatabaseURL  string
	SchoolDomain string
}

func Load() (Config, error) {
	// Existing process variables win over the ignored local .env file.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, errors.New("cannot read local environment file")
	}
	return FromEnv(os.Getenv)
}

func FromEnv(get func(string) string) (Config, error) {
	c := Config{Environment: get("APP_ENV"), HTTPAddress: get("HTTP_ADDR"), DatabaseURL: get("DATABASE_URL"), SchoolDomain: get("SCHOOL_EMAIL_DOMAIN")}
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.Environment != "development" && c.Environment != "production" && c.Environment != "test" {
		return c, errors.New("invalid APP_ENV")
	}
	if c.HTTPAddress == "" {
		c.HTTPAddress = "127.0.0.1:8080"
	}
	if _, _, err := net.SplitHostPort(c.HTTPAddress); err != nil {
		return c, errors.New("invalid HTTP_ADDR")
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" || u.Path == "" {
		return c, errors.New("DATABASE_URL must be a PostgreSQL connection URL")
	}
	if c.SchoolDomain == "" {
		c.SchoolDomain = "ogrenci.example"
	}
	c.SchoolDomain = strings.ToLower(strings.TrimSpace(c.SchoolDomain))
	if c.Environment == "production" && strings.HasSuffix(c.SchoolDomain, ".example") {
		return c, errors.New("production requires the real school email domain")
	}
	return c, nil
}
