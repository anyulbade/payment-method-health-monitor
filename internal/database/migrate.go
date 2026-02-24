package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog/log"
)

// MigrationsDir is the default migrations source URL.
// Override in tests when CWD differs from project root.
var MigrationsDir = "file://migrations"

func RunMigrations(databaseURL string) error {
	m, err := migrate.New(MigrationsDir, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	// Handle dirty state from previous failed migration
	version, dirty, _ := m.Version()
	if dirty {
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("force version: %w", err)
		}
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	version, dirty, _ = m.Version()
	log.Info().
		Uint("version", version).
		Bool("dirty", dirty).
		Msg("migrations applied")

	return nil
}

func RollbackMigrations(databaseURL string) error {
	m, err := migrate.New(MigrationsDir, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	// Handle dirty state before rollback
	version, dirty, _ := m.Version()
	if dirty {
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("force version: %w", err)
		}
	}

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("rollback migrations: %w", err)
	}

	log.Info().Msg("all migrations rolled back")
	return nil
}
