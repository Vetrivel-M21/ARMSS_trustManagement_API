package database

import (
	"database/sql"
	"errors"
	"fmt"

	"trust-management/backend/internal/config"

	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog/log"
)

// RunVersionedMigrations applies backend/migrations/*.sql via golang-migrate.
// This is the authoritative schema-management mechanism; AutoMigrate is not used
// (spec section 62 — never rely only on automatic schema migration).
//
// A dedicated connection with multiStatements enabled is used so each .sql file
// (which contains several CREATE TABLE statements) can run in one Exec.
func RunVersionedMigrations(cfg *config.Config) error {
	dsn := cfg.DBUser + ":" + cfg.DBPassword + "@tcp(" + cfg.DBHost + ":" + cfg.DBPort + ")/" + cfg.DBName + "?charset=utf8mb4&multiStatements=true"
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("opening migration connection: %w", err)
	}
	defer sqlDB.Close()

	driver, err := mysqlmigrate.WithInstance(sqlDB, &mysqlmigrate.Config{})
	if err != nil {
		return fmt.Errorf("initializing migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "mysql", driver)
	if err != nil {
		return fmt.Errorf("initializing migrator: %w", err)
	}
	defer m.Close()

	_, dirty, verErr := m.Version()
	if errors.Is(verErr, migrate.ErrNilVersion) {
		// No migration has ever run against this database. If the core schema
		// already exists (a database that was previously managed by
		// GORM AutoMigrate), baseline it to version 1 without re-running the
		// CREATE TABLE statements, which would fail on tables that already exist.
		var tableExists int
		if err := sqlDB.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'users'",
		).Scan(&tableExists); err != nil {
			return fmt.Errorf("checking for existing schema: %w", err)
		}
		if tableExists > 0 {
			log.Info().Msg("Existing schema detected — baselining migration version to 1 without re-running SQL")
			return m.Force(1)
		}
	} else if verErr != nil {
		return fmt.Errorf("reading migration version: %w", verErr)
	}

	if dirty {
		return fmt.Errorf("database schema is in a dirty migration state — manual intervention required (check schema_migrations table)")
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}
