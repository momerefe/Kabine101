package migrations

import (
	"errors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/momerefe/Kabine101/db"
)

func Open(databaseURL string) (*migrate.Migrate, error) {
	source, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return nil, err
	}
	migration, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		_ = source.Close()
	}
	return migration, err
}

func Up(databaseURL string) error {
	m, err := Open(databaseURL)
	if err != nil {
		return errors.New("migration connection failed; check database settings")
	}
	defer m.Close()
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
