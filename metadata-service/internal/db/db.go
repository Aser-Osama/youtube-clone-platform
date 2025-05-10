package db

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	sqlc "youtube-clone-platform/metadata-service/internal/db/sqlc"

	_ "github.com/mattn/go-sqlite3"
)

// Store encapsulates the SQLC queries and DB connection
type Store struct {
	*sqlc.Queries
	db *sql.DB
}

// New creates a new database connection and runs migrations
func New(dbPath string) (*Store, error) {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runMigrations(database); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create SQLC queries
	queries := sqlc.New(database)

	return &Store{
		Queries: queries,
		db:      database,
	}, nil
}

// runMigrations executes the SQL schema file
func runMigrations(db *sql.DB) error {
	// Read the schema file
	schemaPath := filepath.Join("internal", "db", "schema.sql")
	schema, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute the schema
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}
