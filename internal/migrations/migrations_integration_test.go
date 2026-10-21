//go:build integration

package migrations

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/momerefe/Kabine101/db"
	"github.com/momerefe/Kabine101/internal/database"
)

func TestFreshSchemaUpgradeAndConstraints(t *testing.T) {
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Fatal("TEST_DATABASE_URL is required for explicitly requested integration tests")
	}
	u, err := url.Parse(raw)
	if err != nil || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("integration tests require a dedicated database ending in _test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, raw)
	if err != nil {
		t.Fatal("cannot connect to test database")
	}
	defer admin.Close(context.Background())
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	params := u.Query()
	params.Set("search_path", schema)
	u.RawQuery = params.Encode()
	m, err := Open(u.String())
	if err != nil {
		t.Fatal("cannot open test migration")
	}
	defer m.Close()
	if err = m.Steps(1); err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(ctx, u.String())
	if err != nil {
		t.Fatal("cannot connect to test schema")
	}
	defer conn.Close(context.Background())
	var businessID string
	err = conn.QueryRow(ctx, "INSERT INTO businesses(name, category) VALUES ('Test işletmesi', 'food') RETURNING id").Scan(&businessID)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Up(); err != nil {
		t.Fatal(err)
	}
	version, dirty, err := m.Version()
	if err != nil || dirty || version != db.SchemaVersion {
		t.Fatalf("unexpected schema: %d %t %v", version, dirty, err)
	}
	if err = m.Up(); err != migrate.ErrNoChange {
		t.Fatalf("repeat migration: %v", err)
	}
	var count int
	if err = conn.QueryRow(ctx, "SELECT count(*) FROM businesses WHERE id=$1", businessID).Scan(&count); err != nil || count != 1 {
		t.Fatal("upgrade lost existing data")
	}
	name, err := database.New(conn).GetPlatformName(ctx)
	if err != nil || name != "KoçBenefits" {
		t.Fatal("generated query failed")
	}
	for _, query := range []string{
		"INSERT INTO benefits(business_id,title,method,valid_from,valid_until) VALUES ($1,'Bad dates','code','2026-12-31','2026-01-01')",
		"INSERT INTO benefits(business_id,title,method,publication_status) VALUES ($1,'Unreviewed','code','published')",
		"INSERT INTO benefits(business_id,title,method,target_url) VALUES ($1,'Bad URL','link','javascript:alert(1)')",
	} {
		if _, err = conn.Exec(ctx, query, businessID); err == nil {
			t.Fatal("invalid benefit accepted")
		}
	}
	var accountID, benefitID string
	if err = conn.QueryRow(ctx, "INSERT INTO accounts(name,email,password_hash) VALUES ('Test','test@ogrenci.example','test-only-not-a-password') RETURNING id").Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	if err = conn.QueryRow(ctx, "INSERT INTO benefits(business_id,title,method) VALUES ($1,'Draft','code') RETURNING id", businessID).Scan(&benefitID); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Exec(ctx, "INSERT INTO favorites(account_id,benefit_id) VALUES ($1,$2)", accountID, benefitID); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Exec(ctx, "INSERT INTO favorites(account_id,benefit_id) VALUES ($1,$2)", accountID, benefitID); err == nil {
		t.Fatal("duplicate favorite accepted")
	}
	if _, err = conn.Exec(ctx, "INSERT INTO favorites(account_id,benefit_id) VALUES (gen_random_uuid(),$1)", benefitID); err == nil {
		t.Fatal("orphan favorite accepted")
	}
	if _, err = conn.Exec(ctx, "DELETE FROM accounts WHERE id=$1", accountID); err != nil {
		t.Fatal(err)
	}
	if err = conn.QueryRow(ctx, "SELECT count(*) FROM favorites").Scan(&count); err != nil || count != 0 {
		t.Fatal("account deletion left favorites behind")
	}
}
