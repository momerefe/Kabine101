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
