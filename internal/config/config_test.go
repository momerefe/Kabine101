package config

import (
	"strings"
	"testing"
)

func TestConfigFailsWithoutDatabaseAndDoesNotLeakSecrets(t *testing.T) {
	for _, raw := range []string{"", "http://user:secret@db/test", "postgres://user:secret@%broken/test"} {
		_, err := FromEnv(func(key string) string {
			if key == "DATABASE_URL" {
				return raw
			}
			return ""
		})
		if err == nil {
			t.Fatal("invalid configuration accepted")
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatal("configuration error leaked a credential")
		}
	}
}

func TestProductionRejectsPlaceholderDomain(t *testing.T) {
	_, err := FromEnv(func(key string) string {
		return map[string]string{"DATABASE_URL": "postgres://localhost/test", "APP_ENV": "production"}[key]
	})
	if err == nil {
		t.Fatal("placeholder school domain accepted in production")
	}
}

func TestProductionRequiresExplicitManualPolicyHTTPSAndSMTP(t *testing.T) {
	base := map[string]string{"DATABASE_URL": "postgres://localhost/test", "APP_ENV": "production", "SCHOOL_EMAIL_DOMAIN": "stu.koc.k12.tr", "APP_ORIGIN": "https://benefits.school.test", "ELIGIBILITY_POLICY": "manual", "LOCAL_EMAIL": "false", "SMTP_HOST": "smtp.school.test", "SMTP_PORT": "587", "SMTP_FROM": "noreply@school.test", "SMTP_USERNAME": "sender", "SMTP_PASSWORD": "test-not-a-real-secret"}
	if _, err := FromEnv(func(key string) string { return base[key] }); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"APP_ORIGIN", "ELIGIBILITY_POLICY", "SMTP_HOST", "SMTP_FROM", "SMTP_USERNAME", "SMTP_PASSWORD"} {
		t.Run(key, func(t *testing.T) {
			_, err := FromEnv(func(k string) string {
				if k == key {
					return ""
				}
				return base[k]
			})
			if err == nil {
				t.Fatal("production accepted incomplete configuration")
			}
			if strings.Contains(err.Error(), "test-not-a-real-secret") {
				t.Fatal("configuration leaked secret")
			}
		})
	}
	for key, value := range map[string]string{"APP_ORIGIN": "http://benefits.school.test", "ELIGIBILITY_POLICY": "school_email", "LOCAL_EMAIL": "true", "SCHOOL_EMAIL_DOMAIN": "user@stu.koc.k12.tr", "SMTP_PORT": "0"} {
		if _, err := FromEnv(func(k string) string {
			if k == key {
				return value
			}
			return base[k]
		}); err == nil {
			t.Fatalf("accepted invalid %s", key)
		}
	}
}

func TestDomainValidationRejectsURLsWildcardsAndEmptyLabels(t *testing.T) {
	for _, domain := range []string{"", "localhost", "*.school.test", "https://school.test", "school..test", "-school.test", "school-.test", "school.test:443", "school.test/path"} {
		if ValidDomain(domain) {
			t.Fatalf("accepted invalid domain %q", domain)
		}
	}
	if !ValidDomain("stu.koc.k12.tr") {
		t.Fatal("rejected school domain")
	}
}
