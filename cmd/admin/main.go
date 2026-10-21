package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/momerefe/Kabine101/internal/auth"
	"github.com/momerefe/Kabine101/internal/config"
	"github.com/momerefe/Kabine101/internal/mailer"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run() error {
	name := flag.String("name", "", "new administrator name")
	email := flag.String("email", "", "new administrator school email")
	username := flag.String("username", "", "new email-free operator username")
	flag.Parse()
	*email, *username = strings.TrimSpace(*email), strings.TrimSpace(*username)
	if strings.TrimSpace(*name) == "" || (*email == "") == (*username == "") || flag.NArg() != 0 {
		return errors.New("usage: admin -name NAME (-email SCHOOL_EMAIL | -username USERNAME); password is read privately or from stdin")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	sender, err := mailer.New(cfg)
	if err != nil {
		return err
	}
	minimum := 15
	if *username != "" {
		minimum = 8
	}
	password, err := readPassword(minimum)
	if err != nil {
		return err
	}
	defer clear(password)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return errors.New("cannot initialize database")
	}
	defer pool.Close()
	s, err := auth.New(pool, cfg, sender)
	if err != nil {
		return err
	}
	if *username != "" {
		if err = s.BootstrapOperator(ctx, *name, *username, string(password)); err != nil {
			return errors.New("operator could not be created; check username, password policy and database availability; existing accounts are never modified")
		}
		fmt.Fprintln(os.Stdout, "Operator created. Sign in with the administrator username form.")
		return nil
	}
	if err = s.BootstrapAdmin(ctx, *name, *email, string(password)); err != nil {
		return errors.New("administrator could not be created or verification mail could not be sent; existing accounts are never promoted; if created, request a new verification mail")
	}
	fmt.Fprintln(os.Stdout, "Administrator created. Email verification is required before login.")
	return nil
}

func readPassword(minimum int) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprintf(os.Stderr, "New password (%d–128 characters): ", minimum)
		value, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return nil, errors.New("cannot read password")
		}
		fmt.Fprint(os.Stderr, "Repeat password: ")
		repeated, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		defer clear(repeated)
		if err != nil || string(value) != string(repeated) {
			clear(value)
			return nil, errors.New("password confirmation does not match")
		}
		return value, nil
	}
	// Bound piped input and accept exactly one password line. It never appears
	// in process arguments, shell history (unless supplied there), or logs.
	body, err := io.ReadAll(io.LimitReader(bufio.NewReader(os.Stdin), 1025))
	if err != nil || len(body) > 1024 {
		return nil, errors.New("cannot read password input")
	}
	text := strings.TrimSuffix(strings.TrimSuffix(string(body), "\n"), "\r")
	clear(body)
	if strings.ContainsAny(text, "\r\n") {
		return nil, errors.New("stdin must contain one password line")
	}
	return []byte(text), nil
}
