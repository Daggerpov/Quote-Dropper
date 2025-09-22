package main

import (
	"crypto/subtle"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminRoutes sets up all admin-related routes
func AdminRoutes(r *gin.Engine, db *sql.DB) {
	// Get admin credentials from environment
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	// Admin page to manage unapproved quotes
	r.GET("/admin", BasicAuth(adminUsername, adminPassword), func(c *gin.Context) {
		unapprovedQuotes, err := getUnapprovedQuotes(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch unapproved quotes"})
			return
		}
		c.HTML(http.StatusOK, "admin.html.tmpl", gin.H{"quotes": unapprovedQuotes})
	})

	// Approve a quote
	r.POST("/admin/approve/:id", BasicAuth(adminUsername, adminPassword), handleApproveQuote(db))

	// Dismiss (delete) a quote
	r.POST("/admin/dismiss/:id", BasicAuth(adminUsername, adminPassword), handleDismissQuote(db))

	// Search quotes by keyword
	r.GET("/admin/search/:keyword", handleSearchQuotes(db))

	// Search quotes by author
	r.GET("/admin/search/author/:author", BasicAuth(adminUsername, adminPassword), handleSearchByAuthor(db))

	// Edit a quote
	r.POST("/admin/edit/:id", BasicAuth(adminUsername, adminPassword), handleEditQuote(db))

	// View feedback
	r.GET("/admin/feedback", BasicAuth(adminUsername, adminPassword), handleViewFeedback(db))

	// Delete feedback
	r.DELETE("/admin/feedback/:id", BasicAuth(adminUsername, adminPassword), handleDeleteFeedback(db))
}

// BasicAuth middleware function
func BasicAuth(username, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, ok := c.Request.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			c.Header("WWW-Authenticate", "Basic realm=Restricted")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized access"})
			return
		}
		c.Next()
	}
}

// getUnapprovedQuotes fetches all unapproved quotes from the database
func getUnapprovedQuotes(db *sql.DB) ([]quote, error) {
	rows, err := db.Query("SELECT id, text, author, classification, likes, submitter_name, created_at, updated_at FROM quotes WHERE approved = false ORDER BY created_at DESC")
	if err != nil {
		log.Println("Error fetching unapproved quotes:", err)
		return nil, err
	}
	defer rows.Close()

	return scanQuotesWithSubmitterAndTimestamps(rows)
}

// handleApproveQuote creates a handler for approving quotes
func handleApproveQuote(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		_, err = db.Exec("UPDATE quotes SET approved = true, updated_at = CURRENT_TIMESTAMP WHERE id = $1", id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to approve the quote."})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Quote approved successfully."})
	}
}

// handleDismissQuote creates a handler for dismissing (deleting) quotes
func handleDismissQuote(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		_, err = db.Exec("DELETE FROM quotes WHERE id = $1", id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to dismiss the quote."})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Quote dismissed successfully."})
	}
}

// handleSearchQuotes creates a handler for searching quotes by keyword
func handleSearchQuotes(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyword := c.Param("keyword")
		category := c.Query("category") // Optional category parameter

		// SQL query with optional category filter
		query := "SELECT id, text, author, classification, likes, submitter_name, created_at, updated_at FROM quotes WHERE text ILIKE '%' || $1 || '%'"

		// If category is provided, add it to the WHERE clause
		if category != "" && category != "all" {
			query += " AND classification = $2"
		}

		query += " LIMIT 5"

		var rows *sql.Rows
		var err error

		// Execute query with or without the category parameter
		if category != "" && category != "all" {
			rows, err = db.Query(query, keyword, category)
		} else {
			rows, err = db.Query(query, keyword)
		}

		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to search quotes from the database."})
			return
		}
		defer rows.Close()

		quotes, err := scanQuotesWithSubmitterAndTimestamps(rows)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Error processing search results."})
			return
		}

		c.IndentedJSON(http.StatusOK, quotes)
	}
}

// handleSearchByAuthor creates a handler for searching quotes by author
func handleSearchByAuthor(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		author := c.Param("author")
		// Replace hyphens with spaces and convert to lowercase
		author = strings.ReplaceAll(strings.ToLower(author), "-", " ")

		rows, err := db.Query("SELECT id, text, author, classification, likes, submitter_name, created_at, updated_at FROM quotes WHERE lower(author) LIKE '%' || $1 || '%'", author)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to search quotes from the database."})
			return
		}
		defer rows.Close()

		quotes, err := scanQuotesWithSubmitterAndTimestamps(rows)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Error processing search results."})
			return
		}

		c.IndentedJSON(http.StatusOK, quotes)
	}
}

// handleEditQuote creates a handler for editing quotes
func handleEditQuote(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		var editedQuote quote
		if err := c.ShouldBindJSON(&editedQuote); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid request body."})
			return
		}

		// Check if edited quote text already exists
		var existingID int
		err = db.QueryRow("SELECT id FROM quotes WHERE text = $1 AND id != $2", editedQuote.EditText, id).Scan(&existingID)
		if err == nil {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"message": "Edited quote text already exists in the database."})
			return
		} else if err != sql.ErrNoRows {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to check if edited quote already exists in the database."})
			return
		}

		// Update the quote
		_, err = db.Exec("UPDATE quotes SET text = $1, author = $2, classification = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4",
			editedQuote.EditText, editedQuote.EditAuthor, editedQuote.EditClassification, id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to update the quote."})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Quote updated successfully."})
	}
}

// handleViewFeedback creates a handler for viewing feedback
func handleViewFeedback(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query("SELECT id, type, name, content, image_path, created_at FROM feedback ORDER BY created_at DESC")
		if err != nil {
			log.Println("Error fetching feedback:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feedback"})
			return
		}
		defer rows.Close()

		feedbackItems := []feedback{}

		for rows.Next() {
			var f feedback
			var name sql.NullString
			if err := rows.Scan(&f.ID, &f.Type, &name, &f.Content, &f.ImagePath, &f.CreatedAt); err != nil {
				log.Println("Error scanning feedback row:", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process feedback data"})
				return
			}

			if name.Valid {
				f.Name = name.String
			} else {
				f.Name = ""
			}

			feedbackItems = append(feedbackItems, f)
		}

		if err := rows.Err(); err != nil {
			log.Println("Error iterating feedback rows:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		c.JSON(http.StatusOK, feedbackItems)
	}
}

// handleDeleteFeedback creates a handler for deleting feedback
func handleDeleteFeedback(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid feedback ID."})
			return
		}

		// Get image path before deleting
		var imagePath string
		err = db.QueryRow("SELECT image_path FROM feedback WHERE id = $1", id).Scan(&imagePath)

		// If there's an image, attempt to delete it
		if err == nil && imagePath != "" {
			_ = os.Remove(imagePath)
		}

		// Delete the feedback from the database
		_, err = db.Exec("DELETE FROM feedback WHERE id = $1", id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete feedback."})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Feedback deleted successfully."})
	}
}
