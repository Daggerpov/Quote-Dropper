// Helper to handle error responses
func errorResponse(c *gin.Context, code int, message string) {
	log.Println(message)
	c.JSON(code, gin.H{"error": message})
}

