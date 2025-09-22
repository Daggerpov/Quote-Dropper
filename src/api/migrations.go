package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Migration represents a database migration
type Migration struct {
	ID          int
	Name        string
	SQL         string
	ExecutedAt  *time.Time
	Description string
}

// createMigrationsTable creates the migrations tracking table if it doesn't exist
func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			sql_content TEXT NOT NULL,
			executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			description TEXT
		)
	`)
	return err
}

// getMigrations returns all available migrations in order
func getMigrations() []Migration {
	return []Migration{
		{
			ID:          1,
			Name:        "001_create_feedback_table",
			Description: "Create feedback table for user feedback submissions",
			SQL: `
				CREATE TABLE IF NOT EXISTS feedback (
					id SERIAL PRIMARY KEY,
					type VARCHAR(50) NOT NULL,
					name VARCHAR(100),
					content TEXT NOT NULL,
					image_path VARCHAR(255),
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				)
			`,
		},
		{
			ID:          2,
			Name:        "002_add_submitter_name_to_quotes",
			Description: "Add submitter_name column to quotes table",
			SQL: `
				ALTER TABLE quotes 
				ADD COLUMN IF NOT EXISTS submitter_name VARCHAR(100)
			`,
		},
		{
			ID:          3,
			Name:        "003_add_timestamps_to_quotes",
			Description: "Add created_at and updated_at columns to quotes table",
			SQL: `
				ALTER TABLE quotes 
				ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			`,
		},
		{
			ID:          4,
			Name:        "004_nullify_legacy_quote_timestamps",
			Description: "Set timestamps to null for existing quotes without submitter names",
			SQL: `
				UPDATE quotes 
				SET created_at = NULL, updated_at = NULL 
				WHERE created_at IS NOT NULL 
				AND updated_at IS NOT NULL 
				AND submitter_name IS NULL
			`,
		},
	}
}

// hasBeenExecuted checks if a migration has already been executed
func hasBeenExecuted(db *sql.DB, migrationName string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = $1", migrationName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// executeMigration executes a single migration and records it
func executeMigration(db *sql.DB, migration Migration) error {
	// Check if already executed
	executed, err := hasBeenExecuted(db, migration.Name)
	if err != nil {
		return fmt.Errorf("error checking if migration %s has been executed: %v", migration.Name, err)
	}

	if executed {
		log.Printf("Migration %s already executed, skipping", migration.Name)
		return nil
	}

	// Execute the migration SQL
	log.Printf("Executing migration: %s - %s", migration.Name, migration.Description)
	result, err := db.Exec(migration.SQL)
	if err != nil {
		return fmt.Errorf("error executing migration %s: %v", migration.Name, err)
	}

	// For UPDATE statements, log how many rows were affected
	if rowsAffected, err := result.RowsAffected(); err == nil && rowsAffected > 0 {
		log.Printf("Migration %s affected %d rows", migration.Name, rowsAffected)
	}

	// Record the migration as executed
	_, err = db.Exec(
		"INSERT INTO migrations (id, name, sql_content, description) VALUES ($1, $2, $3, $4)",
		migration.ID, migration.Name, migration.SQL, migration.Description,
	)
	if err != nil {
		return fmt.Errorf("error recording migration %s: %v", migration.Name, err)
	}

	log.Printf("Migration %s completed successfully", migration.Name)
	return nil
}

// runMigrations executes all pending migrations
func runMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("error creating migrations table: %v", err)
	}

	// Get all migrations
	migrations := getMigrations()

	// Execute each migration
	for _, migration := range migrations {
		if err := executeMigration(db, migration); err != nil {
			return err
		}
	}

	log.Println("All migrations completed successfully")
	return nil
}

// getMigrationStatus returns the status of all migrations
func getMigrationStatus(db *sql.DB) ([]Migration, error) {
	if err := createMigrationsTable(db); err != nil {
		return nil, fmt.Errorf("error creating migrations table: %v", err)
	}

	migrations := getMigrations()
	var status []Migration

	for _, migration := range migrations {
		executed, err := hasBeenExecuted(db, migration.Name)
		if err != nil {
			return nil, fmt.Errorf("error checking migration status: %v", err)
		}

		migration.ExecutedAt = nil
		if executed {
			var executedAt time.Time
			err := db.QueryRow("SELECT executed_at FROM migrations WHERE name = $1", migration.Name).Scan(&executedAt)
			if err == nil {
				migration.ExecutedAt = &executedAt
			}
		}

		status = append(status, migration)
	}

	return status, nil
}
