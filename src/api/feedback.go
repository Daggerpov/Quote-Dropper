package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupFeedbackRoutes configures all feedback-related routes
func SetupFeedbackRoutes(r *gin.Engine, db *sql.DB) {
	// Render feedback submission page
	r.GET("/submit-feedback", func(c *gin.Context) {
		c.HTML(http.StatusOK, "feedback.html.tmpl", gin.H{})
	})

	// Handle feedback submission
	r.POST("/submit-feedback", handleSubmitFeedback(db))
}

// handleSubmitFeedback returns a handler for submitting feedback
func handleSubmitFeedback(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get form values
		feedbackType := c.PostForm("type")
		name := c.PostForm("name")
		content := c.PostForm("content")

		// Validate feedback content
		if content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback content is required"})
			return
		}

		// Validate feedback type
		if feedbackType != "general" && feedbackType != "bug" && feedbackType != "feature" {
			feedbackType = "general" // Default to general if invalid type
		}

		// Handle image upload if provided
		var imagePath string
		file, header, err := c.Request.FormFile("image")
		if err == nil && header != nil {
			// Generate a unique filename
			filename := fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename)
			filepath := filepath.Join("uploads", filename)

			// Create the file
			out, err := os.Create(filepath)
			if err != nil {
				log.Println("Error creating file:", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
				return
			}
			defer out.Close()

			// Copy the uploaded file to the destination file
			_, err = io.Copy(out, file)
			if err != nil {
				log.Println("Error copying file:", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
				return
			}

			imagePath = filepath
		}

		// Insert feedback into database
		var id int
		err = db.QueryRow(
			"INSERT INTO feedback (type, name, content, image_path) VALUES ($1, $2, $3, $4) RETURNING id",
			feedbackType, name, content, imagePath,
		).Scan(&id)

		if err != nil {
			log.Println("Error inserting feedback:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feedback"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
	}
}
