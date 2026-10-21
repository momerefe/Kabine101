// Package mailer sends account messages through SMTP. Local mode is restricted
// to a loopback capture server; remote SMTP always requires verified STARTTLS.
package mailer

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/momerefe/Kabine101/internal/config"
)

type Sender struct {
	host, address, username, password, from string
	local                                   bool
}

func New(cfg config.Config) (*Sender, error) {
	if cfg.SMTPHost == "" || strings.ContainsAny(cfg.SMTPHost, "\r\n") {
		return nil, errors.New("SMTP_HOST is required")
	}
	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		return nil, errors.New("invalid SMTP_PORT")
	}
	from, err := mailbox(cfg.SMTPFrom)
	if err != nil {
		return nil, errors.New("valid SMTP_FROM is required")
	}
	if cfg.LocalEmail && (cfg.Environment == "production" || !loopback(cfg.SMTPHost)) {
		return nil, errors.New("local mail capture requires a non-production loopback server")
	}
	return &Sender{host: cfg.SMTPHost, address: net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort)), username: cfg.SMTPUsername, password: cfg.SMTPPassword, from: from, local: cfg.LocalEmail}, nil
}

func loopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func mailbox(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", errors.New("invalid mail address")
	}
	a, err := mail.ParseAddress(value)
	if err != nil || a.Address != value {
		return "", errors.New("invalid mail address")
	}
	return a.Address, nil
}

// Send deliberately returns no SMTP response, address, or message content in
// errors; callers may safely report failure without exposing verification URLs.
func (s *Sender) Send(ctx context.Context, to, subject, body string) error {
	if _, err := mailbox(to); err != nil || strings.ContainsAny(subject, "\r\n") {
		return errors.New("invalid email message")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", s.address)
	if err != nil {
		return errors.New("email delivery unavailable")
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return errors.New("email delivery unavailable")
	}
	defer c.Close()
	if !s.local {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("SMTP requires STARTTLS")
		}
		if err = c.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			return errors.New("SMTP TLS failed")
		}
	}
	if s.username != "" {
		if err = c.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			return errors.New("SMTP authentication failed")
		}
	}
	if err = c.Mail(s.from); err != nil {
		return errors.New("email delivery unavailable")
	}
	if err = c.Rcpt(to); err != nil {
		return errors.New("email delivery unavailable")
	}
	w, err := c.Data()
	if err != nil {
		return errors.New("email delivery unavailable")
	}
	content := message(s.from, to, subject, body)
	if _, err = w.Write([]byte(content)); err != nil {
		_ = w.Close()
		return errors.New("email delivery unavailable")
	}
	if err = w.Close(); err != nil {
		return errors.New("email delivery unavailable")
	}
	// DATA acceptance is the delivery handoff. A failed QUIT cannot undo it.
	_ = c.Quit()
	return nil
}

func message(from, to, subject, body string) string {
	var out strings.Builder
	out.WriteString("From: " + from + "\r\nTo: " + to + "\r\nSubject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	out.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	for len(encoded) > 76 {
		out.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	out.WriteString(encoded + "\r\n")
	return out.String()
}
