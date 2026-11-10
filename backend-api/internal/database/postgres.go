package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

//go:embed init.sql
var initSQL string

// Config holds postgres connection parameters.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Connect(cfg Config) (*sql.DB, error) {
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}

// InitDatabase executes schema initialization from init.sql to ensure all tables and indexes exist.
func InitDatabase(db *sql.DB) error {
	if _, err := db.Exec(initSQL); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}
	return nil
}
