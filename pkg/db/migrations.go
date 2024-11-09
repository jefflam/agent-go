package db

import (
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/sirupsen/logrus"
)

// RunMigrations executes database migrations
func RunMigrations(logger *logrus.Logger, projectRoot string) error {
	migrationsPath := fmt.Sprintf("file://%s", filepath.Join(projectRoot, "migrations"))
	dbURL := constructDBURL()

	logger.WithFields(logrus.Fields{
		"migrations_path": migrationsPath,
		"project_root":    projectRoot,
	}).Debug("Running database migrations")

	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// MigrationStatus returns the current migration version and dirty state
func MigrationStatus(logger *logrus.Logger) (uint, bool, error) {
	logger.Debug("Checking migration status")

	projectRoot, err := findProjectRoot()
	if err != nil {
		return 0, false, fmt.Errorf("failed to find project root: %w", err)
	}

	migrationsPath := fmt.Sprintf("file://%s", filepath.Join(projectRoot, "migrations"))
	dbURL := constructDBURL()

	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"version": version,
		"dirty":   dirty,
	}).Debug("Migration status retrieved")

	return version, dirty, nil
}
