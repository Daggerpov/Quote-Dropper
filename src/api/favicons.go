package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// SetupFavicons configures routes for serving favicon files
func SetupFavicons(r *gin.Engine) {
	// Define possible paths to check for the favicon
	possiblePaths := []string{
		"./templates/images/small-droplet-icon.jpeg",
		"../templates/images/small-droplet-icon.jpeg",
		"./src/api/templates/images/small-droplet-icon.jpeg",
	}

	// Find the first path that exists
	var faviconPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			faviconPath = path
			break
		}
	}

	if faviconPath == "" {
		log.Println("Warning: Favicon file not found in any expected location")
		// Use the last path as fallback
		faviconPath = possiblePaths[len(possiblePaths)-1]
	}

	// Serve favicon.ico from the determined path
	r.GET("/favicon.ico", func(c *gin.Context) {
		// Try serving from the static route we added
		c.Redirect(301, "/templates/images/small-droplet-icon.jpeg")
	})

	// Also add a fallback if direct access is needed
	r.GET("/favicon", func(c *gin.Context) {
		c.File(faviconPath)
	})
}
