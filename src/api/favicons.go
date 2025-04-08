package main

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SetupFavicons configures routes for serving favicon files
func SetupFavicons(r *gin.Engine) {
	// Serve favicon.ico directly from the images directory
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File(filepath.Join("../images", "small-droplet-icon.png"))
	})
}
