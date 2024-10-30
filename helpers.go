// Helper to handle error responses
func errorResponse(c *gin.Context, code int, message string) {
	log.Println(message)
	c.JSON(code, gin.H{"error": message})
}

// Helper to parse integer parameters and validate them
func parseIntParam(c *gin.Context, paramName string) (int, bool) {
	param := c.Param(paramName)
	value, err := strconv.Atoi(param)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "Invalid "+paramName)
		return 0, false
	}
	return value, true
}
