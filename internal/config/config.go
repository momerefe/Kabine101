package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment       string
	HTTPAddress       string
	DatabaseURL       string
	SchoolDomain      string
	AppOrigin         string
	EligibilityPolicy string
	RegistrationOpen  bool
	LocalEmail        bool
	SMTPHost          string
	SMTPPort          int
	SMTPUsername      string
	SMTPPassword      string
	SMTPFrom          string
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
	if !ValidDomain(c.SchoolDomain) {
		return c, errors.New("invalid SCHOOL_EMAIL_DOMAIN")
	}
	if c.Environment == "production" && (strings.HasSuffix(c.SchoolDomain, ".example") || c.SchoolDomain == "example.com" || c.SchoolDomain == "example.org" || c.SchoolDomain == "example.net") {
		return c, errors.New("production requires the real school email domain")
	}
	c.AppOrigin = strings.TrimRight(get("APP_ORIGIN"), "/")
	if c.AppOrigin == "" && c.Environment != "production" {
		c.AppOrigin = "http://127.0.0.1:5173"
	}
	o, err := url.Parse(c.AppOrigin)
	if err != nil || o.Host == "" || (o.Scheme != "http" && o.Scheme != "https") || o.User != nil || o.Path != "" || o.RawQuery != "" || o.Fragment != "" || (c.Environment == "production" && o.Scheme != "https") {
		return c, errors.New("APP_ORIGIN must be a trusted origin (HTTPS in production)")
	}
	c.EligibilityPolicy = get("ELIGIBILITY_POLICY")
	if c.EligibilityPolicy == "" {
		if c.Environment == "production" {
			return c, errors.New("production requires explicit ELIGIBILITY_POLICY")
		}
		c.EligibilityPolicy = "manual"
	}
	if c.EligibilityPolicy != "manual" {
		return c, errors.New("invalid ELIGIBILITY_POLICY")
	}
	c.RegistrationOpen = true
	if raw := get("REGISTRATION_OPEN"); raw != "" {
		c.RegistrationOpen, err = strconv.ParseBool(raw)
		if err != nil {
			return c, errors.New("invalid REGISTRATION_OPEN")
		}
	}
	c.LocalEmail = c.Environment != "production"
	if raw := get("LOCAL_EMAIL"); raw != "" {
		c.LocalEmail, err = strconv.ParseBool(raw)
		if err != nil {
			return c, errors.New("invalid LOCAL_EMAIL")
		}
	}
	c.SMTPHost, c.SMTPUsername, c.SMTPPassword, c.SMTPFrom = get("SMTP_HOST"), get("SMTP_USERNAME"), get("SMTP_PASSWORD"), get("SMTP_FROM")
	c.SMTPPort = 587
	if raw := get("SMTP_PORT"); raw != "" {
		c.SMTPPort, err = strconv.Atoi(raw)
		if err != nil || c.SMTPPort < 1 || c.SMTPPort > 65535 {
			return c, errors.New("invalid SMTP_PORT")
		}
	}
	if c.Environment == "production" && (c.LocalEmail || c.SMTPHost == "" || c.SMTPFrom == "" || c.SMTPUsername == "" || c.SMTPPassword == "") {
		return c, errors.New("production requires authenticated SMTP and LOCAL_EMAIL=false")
	}
	if !c.LocalEmail && (c.SMTPHost == "" || c.SMTPFrom == "") {
		return c, errors.New("SMTP_HOST and SMTP_FROM are required")
	}
	return c, nil
}

var domainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)

func ValidDomain(domain string) bool { return len(domain) <= 253 && domainPattern.MatchString(domain) }
