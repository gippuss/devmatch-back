package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func RunMigrations(databaseURL, migrationDir string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, migrationDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

func RunMigrationCommand(databaseURL, migrationDir, command string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	switch command {
	case "up":
		return goose.Up(db, migrationDir)
	case "down":
		return goose.Down(db, migrationDir)
	case "reset":
		return goose.Reset(db, migrationDir)
	default:
		return fmt.Errorf("unsupported migration command: %s", command)
	}
}
