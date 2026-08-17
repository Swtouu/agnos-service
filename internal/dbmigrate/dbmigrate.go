// Package dbmigrate applies the embedded SQL migrations at process startup.
// Both cmd/api and cmd/seed call Apply so either one works standalone and in
// any order — required for platforms (e.g. Railway) that don't offer a
// separate pre-deploy migration step the way docker-compose's old init
// container did.
package dbmigrate

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/watt-siwat/agnos-backend/migrations"
)

// Apply runs all pending migrations against databaseURL. Idempotent — safe
// to call on every process boot.
func Apply(databaseURL string) error {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return fmt.Errorf("init migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
