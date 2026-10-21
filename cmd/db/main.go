package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/momerefe/Kabine101/internal/config"
	"github.com/momerefe/Kabine101/internal/migrations"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/db [up|version|seed-demo]")
		os.Exit(1)
	}
	cfg, err := config.Load()
	if err != nil {
		fail(err.Error())
	}
	switch os.Args[1] {
	case "up":
		if err := migrations.Up(cfg.DatabaseURL); err != nil {
			fail("Migration failed. Check database availability and schema state; no credentials are printed.")
		}
		fmt.Println("Schema is up to date.")
	case "version":
		m, err := migrations.Open(cfg.DatabaseURL)
		if err != nil {
			fail("Cannot connect to migration database.")
		}
		defer m.Close()
		v, dirty, err := m.Version()
		if err != nil {
			fail("No readable schema version. Run migrations first.")
		}
		fmt.Printf("Schema version: %d; dirty: %t\n", v, dirty)
	case "seed-demo":
		if cfg.Environment != "development" {
			fail("Demo seed is allowed only in development.")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
		if err != nil {
			fail("Cannot connect to development database.")
		}
		defer conn.Close(ctx)
		tx, err := conn.Begin(ctx)
		if err != nil {
			fail("Cannot start seed transaction.")
		}
		defer tx.Rollback(ctx)
		_, err = tx.Exec(ctx, `INSERT INTO businesses (id, name, category, description, partnership_status) VALUES ('10000000-0000-4000-8000-000000000001', 'Mola Coffee (Demo)', 'food', 'Kurgusal geliştirme kaydı; gerçek sponsor değildir.', 'active') ON CONFLICT (id) DO NOTHING`)
		if err != nil {
			fail("Demo business seed failed.")
		}
		_, err = tx.Exec(ctx, `INSERT INTO benefits (id, business_id, title, method, code) VALUES ('20000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', 'Demo kahve avantajı taslağı', 'code', 'DEMO-MOLA20') ON CONFLICT (id) DO NOTHING`)
		if err != nil {
			fail("Demo benefit seed failed.")
		}
		if err = tx.Commit(ctx); err != nil {
			fail("Demo seed could not be saved.")
		}
		fmt.Println("Development demo business and draft created. No accounts or published offers were created.")
	default:
		fail("Unknown database command.")
	}
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
