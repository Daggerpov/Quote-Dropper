package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// runMigrationCLI is a simple CLI tool for managing migrations
func runMigrationCLI() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	// Get database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("DATABASE_URL environment variable must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	switch command {
	case "migrate":
		fmt.Println("Running migrations...")
		err := runMigrations(db)
		if err != nil {
			log.Fatalf("Error running migrations: %v", err)
		}
		fmt.Println("Migrations completed successfully!")

	case "status":
		fmt.Println("Migration Status:")
		fmt.Println("================")
		status, err := getMigrationStatus(db)
		if err != nil {
			log.Fatalf("Error getting migration status: %v", err)
		}

		for _, migration := range status {
			statusStr := "PENDING"
			executedAt := "Not executed"
			if migration.ExecutedAt != nil {
				statusStr = "EXECUTED"
				executedAt = migration.ExecutedAt.Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%-35s | %-10s | %s\n", migration.Name, statusStr, executedAt)
			fmt.Printf("    %s\n\n", migration.Description)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: go run . migrate <command>")
	fmt.Println("       go run migrate.go main.go migrations.go models.go <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  migrate    Run all pending migrations")
	fmt.Println("  status     Show migration status")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  DATABASE_URL    Database connection string (optional)")
}

// Uncomment the main function below to use this as a standalone migration tool
// func main() {
// 	runMigrationCLI()
// }
