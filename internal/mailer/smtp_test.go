package mailer

import (
	"bufio"
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/momerefe/Kabine101/internal/config"
)

func TestLocalCaptureBoundary(t *testing.T) {
	base := config.Config{Environment: "development", LocalEmail: true, SMTPHost: "127.0.0.1", SMTPPort: 1025, SMTPFrom: "noreply@kocbenefits.example"}
	if _, err := New(base); err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"smtp.example.org", "127.0.0.1.evil.example", "0.0.0.0"} {
		cfg := base
		cfg.SMTPHost = host
		if _, err := New(cfg); err == nil {
			t.Fatalf("accepted remote capture host %s", host)
		}
	}
	base.Environment = "production"
	if _, err := New(base); err == nil {
		t.Fatal("local capture accepted in production")
	}
}

func TestSMTPDeliveryAndTLSRequirement(t *testing.T) {
	for _, local := range []bool{true, false} {
		t.Run(map[bool]string{true: "local capture", false: "remote without TLS refused"}[local], func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			captured := make(chan string, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				reader := bufio.NewReader(conn)
				_, _ = io.WriteString(conn, "220 test SMTP\r\n")
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						_, _ = io.WriteString(conn, "250 test\r\n")
					case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
						_, _ = io.WriteString(conn, "250 ok\r\n")
					case strings.HasPrefix(line, "DATA"):
						_, _ = io.WriteString(conn, "354 send message\r\n")
						var b strings.Builder
						for {
							line, err = reader.ReadString('\n')
							if err != nil {
								return
							}
							if line == ".\r\n" {
								break
							}
							b.WriteString(line)
						}
						captured <- b.String()
						_, _ = io.WriteString(conn, "250 accepted\r\n")
					case strings.HasPrefix(line, "QUIT"):
						_, _ = io.WriteString(conn, "221 bye\r\n")
						return
					default:
						_, _ = io.WriteString(conn, "500 unsupported\r\n")
					}
				}
			}()
			s := &Sender{host: "127.0.0.1", address: listener.Addr().String(), from: "noreply@kocbenefits.example", local: local}
			err = s.Send(context.Background(), "student@ogrenci.example", "E-posta doğrulaması", "Doğrulama bağlantısı: http://127.0.0.1/#verify-email?token=sample")
			if !local {
				if err == nil {
					t.Fatal("accepted unencrypted remote SMTP")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case content := <-captured:
				m, err := mail.ReadMessage(strings.NewReader(content))
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, m.Body))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(decoded), "Doğrulama bağlantısı:") || !strings.Contains(string(decoded), "#verify-email?token=sample") {
					t.Fatal("UTF8 body or token fragment lost")
				}
			case <-time.After(time.Second):
				t.Fatal("message not accepted")
			}
		})
	}
}

func TestHeaderInjectionRejected(t *testing.T) {
	s := &Sender{}
	for _, value := range []struct{ to, subject string }{{"a@ogrenci.example\r\nBcc: other@example.org", "Subject"}, {"a@ogrenci.example", "subject\r\nBcc: other@example.org"}} {
		if err := s.Send(context.Background(), value.to, value.subject, "body"); err == nil {
			t.Fatal("accepted header injection")
		}
	}
}

func TestCancellationClosesSMTPConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	done := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		close(accepted)
		_, _ = io.Copy(io.Discard, conn)
		close(done)
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Sender{host: "127.0.0.1", address: listener.Addr().String(), from: "sender@ogrenci.example", local: true}
	result := make(chan error, 1)
	go func() { result <- s.Send(ctx, "recipient@ogrenci.example", "subject", "body") }()
	<-accepted
	cancel()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("cancelled delivery succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP did not honor cancellation")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SMTP connection leaked")
	}
}
