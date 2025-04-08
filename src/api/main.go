package main

import (
	"database/sql"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Set Gin to release mode in production
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("WARNING: DATABASE_URL environment variable not set, running without database")
		dbURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	// Connect to the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Println("Error connecting to database:", err)
		log.Println("Running without database functionality")
		db = nil
	} else {
		defer db.Close()

		// Create feedback table if it doesn't exist
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS feedback (
				id SERIAL PRIMARY KEY,
				type VARCHAR(50) NOT NULL,
				name VARCHAR(100),
				content TEXT NOT NULL,
				image_path VARCHAR(255),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			log.Println("Error creating feedback table:", err)
			log.Println("Running without database functionality")
			db = nil
		}
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize Gin router
	r := gin.Default()

	// Create uploads directory if it doesn't exist
	err = os.MkdirAll("./uploads", 0755)
	if err != nil {
		log.Println("Error creating uploads directory:", err)
		log.Fatal(err)
	}

	// Load templates from the templates directory
	execDir, _ := os.Getwd()
	templatePath := filepath.Join(execDir, "..", "templates")
	t, err := template.ParseGlob(filepath.Join(templatePath, "*.tmpl"))
	if err != nil {
		log.Fatal("Error loading templates:", err)
	}
	r.SetHTMLTemplate(t)

	// Initialize rate limiter: 60 requests per minute
	rateLimiter := NewRateLimiter(60, time.Minute)

	// Apply rate limiting middleware to all routes
	r.Use(RateLimitMiddleware(rateLimiter))

	// Set up static file serving
	r.Static("/static", "./static")
	r.Static("/uploads", "./uploads")
	r.Static("/images", "../images")

	// Set up favicons
	SetupFavicons(r)

	// Set up routes for different components
	SetupQuoteRoutes(r, db)
	SetupFeedbackRoutes(r, db)
	AdminRoutes(r, db)

	// Home route
	r.GET("/", func(c *gin.Context) {
		data := map[string]string{
			"Region": os.Getenv("FLY_REGION"),
		}
		c.HTML(http.StatusOK, "index.html.tmpl", data)
	})

	// Start the server
	log.Println("listening on", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatal(err)
	}
}
