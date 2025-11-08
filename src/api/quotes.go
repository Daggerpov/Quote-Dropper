package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/gin-gonic/gin"
)

// isAuthorValid checks if an author name is valid and should be displayed
// This mirrors the iOS app's isAuthorValid function from AuthorHelper.swift
func isAuthorValid(author string) bool {
	return author != "Unknown Author" &&
		author != "NULL" &&
		author != "" &&
		strings.TrimSpace(author) != ""
}

// isBrowserRequest checks if the request is coming from a web browser
func isBrowserRequest(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	userAgent := c.GetHeader("User-Agent")

	// Check if the request accepts HTML
	acceptsHTML := strings.Contains(accept, "text/html")

	// Check if it's NOT from your mobile app (Quote Droplet)
	// You can customize this logic based on your app's user agent
	isNotMobileApp := !strings.Contains(userAgent, "Quote Droplet") &&
		!strings.Contains(userAgent, "okhttp") && // Common Android HTTP library
		!strings.Contains(userAgent, "CFNetwork") // Common iOS HTTP library

	return acceptsHTML && isNotMobileApp
}

// QuotePageData represents data for the quotes HTML template
type QuotePageData struct {
	Title       string
	Description string
	Quotes      []quote
	Stats       *QuoteStats
}

// QuoteStats represents statistics about the quotes being displayed
type QuoteStats struct {
	Count          int
	MaxLength      int
	Classification string
}

// renderQuotesHTML renders quotes as HTML for browser requests
func renderQuotesHTML(c *gin.Context, quotes []quote, title, description string, stats *QuoteStats) {
	data := QuotePageData{
		Title:       title,
		Description: description,
		Quotes:      quotes,
		Stats:       stats,
	}
	c.HTML(http.StatusOK, "quotes.html.tmpl", data)
}

// SetupQuoteRoutes configures all quote-related routes
func SetupQuoteRoutes(r *gin.Engine, db *sql.DB) {
	// REST API endpoints
	// GET /quotes - Get all quotes with optional query parameters:
	//   ?classification=X - filter by classification
	//   ?author=X - filter by author
	//   ?maxLength=X - filter by max text length
	//   ?from=X - get quotes starting from ID
	//   ?search=X - search quotes by keyword
	//   ?category=X - filter search by category
	//   ?index=X - get specific quote index by author (requires author param)
	r.GET("/quotes", handleGetQuotes(db))

	// GET /quotes/random - Get a random quote
	//   ?classification=X - filter by classification
	//   ?maxLength=X - filter by max text length
	r.GET("/quotes/random", handleGetRandomQuote(db))

	// GET /quotes/recent - Get recent quotes
	//   ?limit=X - number of quotes to return (default: 5, max: 10)
	r.GET("/quotes/recent", handleGetRecentQuotes(db))

	// GET /quotes/top - Get top (most liked) quotes
	//   ?category=X - filter by category
	r.GET("/quotes/top", handleGetTopQuotes(db))

	// GET /quotes/count - Get quote count
	//   ?category=X - filter by category
	r.GET("/quotes/count", handleGetQuoteCount(db))

	// GET /quotes/:id - Get a specific quote by ID
	r.GET("/quotes/:id", handleGetQuoteByID(db))

	// POST /quotes - Create a new quote
	r.POST("/quotes", handleAddQuote(db))

	// POST /quotes/:id/like - Like a quote
	r.POST("/quotes/:id/like", handleLikeQuote(db))

	// DELETE /quotes/:id/like - Unlike a quote
	r.DELETE("/quotes/:id/like", handleUnlikeQuote(db))

	// GET /categories - Get all available categories
	r.GET("/categories", handleGetCategories(db))

	// Web form routes (for browser UI)
	r.GET("/submit-quote", func(c *gin.Context) {
		c.HTML(http.StatusOK, "submit-quote.html.tmpl", gin.H{})
	})
	r.POST("/submit-quote", handleSubmitQuote(db))
}

// handleGetQuotes returns a handler for getting quotes with various query parameters
// Supports: ?classification=X, ?author=X, ?maxLength=X, ?from=X, ?search=X, ?index=X
func handleGetQuotes(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract query parameters
		classification := c.Query("classification")
		author := c.Query("author")
		maxLengthStr := c.Query("maxLength")
		fromStr := c.Query("from")
		search := c.Query("search")
		indexStr := c.Query("index")
		category := c.Query("category") // For search filtering

		// Build query dynamically
		query := "SELECT id, text, author, classification, likes FROM quotes WHERE approved = true"
		args := []interface{}{}
		argCount := 1

		// Add classification filter
		if classification != "" {
			query += " AND classification = $" + strconv.Itoa(argCount)
			args = append(args, classification)
			argCount++
		}

		// Add author filter
		if author != "" {
			query += " AND author = $" + strconv.Itoa(argCount)
			args = append(args, author)
			argCount++
		}

		// Add max length filter
		if maxLengthStr != "" {
			maxLength, err := strconv.Atoi(maxLengthStr)
			if err != nil || maxLength < 0 {
				if isBrowserRequest(c) {
					c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
						Title:       "Invalid Request",
						Description: "Invalid maxLength parameter",
						Quotes:      []quote{},
					})
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid maxLength parameter"})
				}
				return
			}
			query += " AND LENGTH(text) <= $" + strconv.Itoa(argCount)
			args = append(args, maxLength)
			argCount++
		}

		// Add from ID filter
		if fromStr != "" {
			fromID, err := strconv.Atoi(fromStr)
			if err != nil || fromID < 0 {
				if isBrowserRequest(c) {
					c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
						Title:       "Invalid Request",
						Description: "Invalid from parameter",
						Quotes:      []quote{},
					})
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid from parameter"})
				}
				return
			}
			query += " AND id >= $" + strconv.Itoa(argCount)
			args = append(args, fromID)
			argCount++
		}

		// Add search filter
		if search != "" {
			query += " AND (text ILIKE '%' || $" + strconv.Itoa(argCount) + " || '%' OR author ILIKE '%' || $" + strconv.Itoa(argCount) + " || '%')"
			args = append(args, search)
			argCount++

			// Optional category filter for search
			if category != "" && category != "all" {
				query += " AND classification = $" + strconv.Itoa(argCount)
				args = append(args, category)
				argCount++
			}

			// Add ordering and limit for search
			query += " ORDER BY likes DESC, id DESC LIMIT 10"
		} else {
			// Default ordering
			if fromStr != "" {
				query += " ORDER BY id"
			}
			if author != "" {
				query += " ORDER BY id"
			}
		}

		log.Printf("Executing query: %s with args: %v", query, args)

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Failed to fetch quotes",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes"})
			}
			return
		}
		defer rows.Close()

		quotes, err := scanQuotes(rows)
		if err != nil {
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Failed to process quotes",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process quotes"})
			}
			return
		}

		// Handle index parameter (get specific quote by author and index)
		if indexStr != "" && author != "" {
			index, err := strconv.Atoi(indexStr)
			if err != nil || index < 0 {
				if isBrowserRequest(c) {
					c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
						Title:       "Invalid Request",
						Description: "Invalid index parameter",
						Quotes:      []quote{},
					})
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index parameter"})
				}
				return
			}

			if index >= len(quotes) {
				if isBrowserRequest(c) {
					c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
						Title:       "Index Out of Range",
						Description: "The specified index is out of range",
						Quotes:      []quote{},
					})
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Index out of range"})
				}
				return
			}

			if isBrowserRequest(c) {
				stats := &QuoteStats{Count: 1}
				title := "Quote by " + author + " (Index " + indexStr + ")"
				description := "Quote #" + strconv.Itoa(index) + " by " + author
				renderQuotesHTML(c, []quote{quotes[index]}, title, description, stats)
			} else {
				c.IndentedJSON(http.StatusOK, quotes[index])
			}
			return
		}

		// Prepare response based on request type
		if isBrowserRequest(c) {
			stats := &QuoteStats{Count: len(quotes)}
			title := "Quotes"
			description := "Browse quotes"

			if classification != "" {
				stats.Classification = classification
				title = cases.Title(language.English).String(classification) + " Quotes"
				description = "All quotes in the " + classification + " category"
			}
			if author != "" {
				title = "Quotes by " + author
				description = "All quotes attributed to " + author
			}
			if search != "" {
				title = "Search Results"
				description = "Quotes matching \"" + search + "\""
			}
			if maxLengthStr != "" {
				maxLength, _ := strconv.Atoi(maxLengthStr)
				stats.MaxLength = maxLength
			}

			renderQuotesHTML(c, quotes, title, description, stats)
		} else {
			c.IndentedJSON(http.StatusOK, quotes)
		}
	}
}

// handleGetRandomQuote returns a handler for getting a random quote
// Supports: ?classification=X, ?maxLength=X
func handleGetRandomQuote(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		classification := c.Query("classification")
		maxLengthStr := c.Query("maxLength")

		// Build query dynamically
		query := "SELECT id, text, author, classification, likes FROM quotes WHERE approved = true"
		args := []interface{}{}
		argCount := 1

		// Add classification filter
		if classification != "" && classification != "all" {
			query += " AND classification = $" + strconv.Itoa(argCount)
			args = append(args, classification)
			argCount++
		}

		// Add max length filter
		if maxLengthStr != "" {
			maxLength, err := strconv.Atoi(maxLengthStr)
			if err != nil || maxLength < 0 {
				if isBrowserRequest(c) {
					c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
						Title:       "Invalid Request",
						Description: "Invalid maxLength parameter",
						Quotes:      []quote{},
					})
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid maxLength parameter"})
				}
				return
			}
			query += " AND LENGTH(text) <= $" + strconv.Itoa(argCount)
			args = append(args, maxLength)
			argCount++
		}

		log.Printf("Executing query: %s with args: %v", query, args)

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Failed to fetch quotes",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes"})
			}
			return
		}
		defer rows.Close()

		quotes, err := scanQuotes(rows)
		if err != nil {
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Failed to process quotes",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process quotes"})
			}
			return
		}

		if len(quotes) == 0 {
			if isBrowserRequest(c) {
				c.HTML(http.StatusNotFound, "quotes.html.tmpl", QuotePageData{
					Title:       "No Quote Found",
					Description: "No random quote found with the specified criteria",
					Quotes:      []quote{},
				})
			} else {
				c.IndentedJSON(http.StatusNotFound, gin.H{"message": "No random quote found with the specified criteria."})
			}
			return
		}

		// Select a random quote from the results
		randomIndex := rand.Intn(len(quotes))
		randomQuote := quotes[randomIndex]

		if isBrowserRequest(c) {
			stats := &QuoteStats{Count: 1}
			if classification != "" {
				stats.Classification = classification
			}
			title := "Random Quote"
			if classification != "" && classification != "all" {
				title = "Random " + cases.Title(language.English).String(classification) + " Quote"
			}
			description := "A randomly selected quote"
			renderQuotesHTML(c, []quote{randomQuote}, title, description, stats)
		} else {
			c.IndentedJSON(http.StatusOK, randomQuote)
		}
	}
}

// handleGetRecentQuotes returns a handler for getting recent quotes
// Supports: ?limit=X (default: 5, max: 10)
func handleGetRecentQuotes(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "5")

		// Validate the limit parameter
		numLimit, err := strconv.Atoi(limitStr)
		if err != nil || numLimit < 1 || numLimit > 10 {
			if isBrowserRequest(c) {
				c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
					Title:       "Invalid Request",
					Description: "Invalid limit parameter. It should be a number between 1 and 10.",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter. It should be a number between 1 and 10."})
			}
			return
		}

		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE approved = true ORDER BY id DESC LIMIT $1", numLimit)
		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Failed to fetch recent approved quotes",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent approved quotes"})
			}
			return
		}
		defer rows.Close()

		quotes, err := scanQuotes(rows)
		if err != nil {
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Failed to process quotes",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process quotes"})
			}
			return
		}

		if isBrowserRequest(c) {
			stats := &QuoteStats{Count: len(quotes)}
			title := "Recent Quotes"
			description := "The " + limitStr + " most recently added quotes"
			renderQuotesHTML(c, quotes, title, description, stats)
		} else {
			c.JSON(http.StatusOK, quotes)
		}
	}
}

// handleGetQuoteByID returns a handler for getting a specific quote by ID
func handleGetQuoteByID(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			if isBrowserRequest(c) {
				c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
					Title:       "Invalid Request",
					Description: "Invalid quote ID specified",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			}
			return
		}

		var q quote
		err = db.QueryRow("SELECT id, text, author, classification, likes FROM quotes WHERE approved = TRUE AND id = $1", id).Scan(&q.ID, &q.Text, &q.Author, &q.Classification, &q.Likes)
		if err == sql.ErrNoRows {
			if isBrowserRequest(c) {
				c.HTML(http.StatusNotFound, "quotes.html.tmpl", QuotePageData{
					Title:       "Quote Not Found",
					Description: "The requested quote could not be found",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusNotFound, gin.H{"message": "Quote not found."})
			}
			return
		} else if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Database error occurred",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			}
			return
		}

		if isBrowserRequest(c) {
			stats := &QuoteStats{Count: 1}
			title := "Quote #" + idStr
			description := "Individual quote details"
			renderQuotesHTML(c, []quote{q}, title, description, stats)
		} else {
			c.IndentedJSON(http.StatusOK, q)
		}
	}
}

// handleGetQuoteCount returns a handler for getting the number of quotes in a given category
// Supports: ?category=X
func handleGetQuoteCount(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")
		if category == "" || category == "all" {
			// If category is not specified, retrieve the total count of all quotes
			var totalCount int
			err := db.QueryRow("SELECT COUNT(*) FROM quotes WHERE approved = true").Scan(&totalCount)
			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve total quote count from the database."})
				return
			}

			c.IndentedJSON(http.StatusOK, gin.H{"category": "all", "count": totalCount})
			return
		}

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM quotes WHERE classification = $1 AND approved = true", strings.ToLower(category)).Scan(&count)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve quote count from the database."})
			return
		}

		c.IndentedJSON(http.StatusOK, gin.H{"category": category, "count": count})
	}
}

// handleGetTopQuotes returns a handler for getting the top (most liked) quotes
// Supports: ?category=X
func handleGetTopQuotes(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")

		var query string
		var rows *sql.Rows
		var err error

		// Default to returning at most 10 quotes
		limit := 10

		if category == "" || category == "all" {
			// Get top quotes across all categories
			query = "SELECT id, text, author, classification, likes FROM quotes WHERE approved = true ORDER BY likes DESC LIMIT $1"
			rows, err = db.Query(query, limit)
		} else {
			// Get top quotes for a specific category
			query = "SELECT id, text, author, classification, likes FROM quotes WHERE approved = true AND classification = $1 ORDER BY likes DESC LIMIT $2"
			rows, err = db.Query(query, strings.ToLower(category), limit)
		}

		if err != nil {
			log.Println("Error fetching top quotes:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top quotes"})
			return
		}
		defer rows.Close()

		quotes, err := scanQuotes(rows)
		if err != nil {
			log.Println("Error scanning quotes:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process quotes"})
			return
		}

		c.JSON(http.StatusOK, quotes)
	}
}

// handleGetCategories returns a handler for getting all available categories
func handleGetCategories(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query("SELECT DISTINCT classification FROM quotes WHERE approved = true")
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve categories"})
			return
		}
		defer rows.Close()

		// Define valid categories to filter out problematic ones
		validCategories := map[string]bool{
			"wisdom":      true,
			"motivation":  true,
			"discipline":  true,
			"philosophy":  true,
			"inspiration": true,
			"upliftment":  true,
			"love":        true,
		}

		categories := []string{}
		categorySet := make(map[string]bool) // To prevent duplicates

		for rows.Next() {
			var category string
			if err := rows.Scan(&category); err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to process categories"})
				return
			}

			// Filter out invalid categories (blank, null, all, etc.) and duplicates
			if validCategories[category] && !categorySet[category] {
				categories = append(categories, category)
				categorySet[category] = true
			}
		}

		if err := rows.Err(); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to process categories"})
			return
		}

		c.IndentedJSON(http.StatusOK, gin.H{"categories": categories})
	}
}

// handleAddQuote returns a handler for adding a new quote
func handleAddQuote(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse the request body into a new quote struct
		var q quote
		if err := c.ShouldBindJSON(&q); err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid request body."})
			return
		}

		// Validate the incoming quote
		if q.Text == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Text field is required."})
			return
		}

		// Trim trailing spaces after the last character/punctuation mark
		q.Text = strings.TrimSpace(q.Text)

		// Add a period if there isn't any yet in the 'Text' field and it doesn't end with a question mark
		if q.Text[len(q.Text)-1] != '.' && q.Text[len(q.Text)-1] != '?' && q.Text[len(q.Text)-1] != '!' {
			q.Text += "."
		}

		// Check if quote already exists
		var existingID int
		err := db.QueryRow("SELECT id FROM quotes WHERE text=$1", q.Text).Scan(&existingID)
		if err == nil {
			// Quote already exists, return an error message
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"message": "Identical quote already exists in the database."})
			return
		} else if err != sql.ErrNoRows {
			// Error occurred while querying the database
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to check if quote already exists in the database."})
			return
		}

		// Set the approved status to false for new quotes
		q.Approved = false

		// Set likes to a random value between 12 and 37 for new quotes
		q.Likes = rand.Intn(37-12+1) + 12

		// Insert the new quote into the database
		var id int
		if q.Author == "" {
			q.Author = "NULL"
		} else {
			// Create a Title case converter
			converter := cases.Title(language.English)

			// Apply the title case conversion to the author field
			q.Author = converter.String(q.Author)
		}

		if q.Classification == "" {
			q.Classification = "NULL"
		} else {
			q.Classification = strings.ToLower(q.Classification) // Convert classification to lowercase
		}
		err = db.QueryRow("INSERT INTO quotes (text, author, classification, approved, likes, submitter_name, created_at, updated_at) VALUES ($1, $2, LOWER($3), $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id", q.Text, q.Author, q.Classification, q.Approved, q.Likes, q.SubmitterName).Scan(&id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to insert quote into the database."})
			return
		}
		q.ID = id

		// Create a response object
		response := quote{
			ID:             q.ID,
			Text:           q.Text,
			Author:         q.Author,
			Classification: q.Classification,
			Approved:       q.Approved,
			Likes:          q.Likes, // Include likes in the response
			SubmitterName:  q.SubmitterName,
		}

		// Return the newly created quote in the response
		c.IndentedJSON(http.StatusCreated, response)
	}
}

// handleLikeQuote returns a handler for incrementing likes for a quote by ID
func handleLikeQuote(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		// Increment the like count for the quote with the given ID
		err = addLikeToQuote(id, db)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to like the quote."})
			return
		}

		// Fetch the updated quote from the database
		var updatedQuote quote
		err = db.QueryRow("SELECT id, text, author, classification, likes FROM quotes WHERE id = $1 AND approved = true", id).Scan(&updatedQuote.ID, &updatedQuote.Text, &updatedQuote.Author, &updatedQuote.Classification, &updatedQuote.Likes)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch updated quote."})
			return
		}

		// Return the updated quote with incremented like count
		c.IndentedJSON(http.StatusOK, updatedQuote)
	}
}

// handleUnlikeQuote returns a handler for decrementing likes for a quote by ID
func handleUnlikeQuote(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		// Decrement the like count for the quote with the given ID
		err = removeLikeFromQuote(id, db)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to unlike the quote."})
			return
		}

		// Fetch the updated quote from the database
		var updatedQuote quote
		err = db.QueryRow("SELECT id, text, author, classification, likes FROM quotes WHERE id = $1 AND approved = true", id).Scan(&updatedQuote.ID, &updatedQuote.Text, &updatedQuote.Author, &updatedQuote.Classification, &updatedQuote.Likes)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch updated quote."})
			return
		}

		// Return the updated quote with decremented like count
		c.IndentedJSON(http.StatusOK, updatedQuote)
	}
}

// handleSubmitQuote returns a handler for submitting a quote through the web form
func handleSubmitQuote(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get form values
		text := c.PostForm("text")
		author := c.PostForm("author")
		classification := c.PostForm("classification")
		submitterName := c.PostForm("submitter_name")

		// Validate quote text
		if text == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Quote text is required"})
			return
		}

		// Validate classification (category)
		validCategories := map[string]bool{
			"wisdom":      true,
			"motivation":  true,
			"discipline":  true,
			"philosophy":  true,
			"inspiration": true,
			"upliftment":  true,
			"love":        true,
		}

		if !validCategories[classification] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quote category"})
			return
		}

		// Check for duplicate quotes
		var existingID int
		err := db.QueryRow("SELECT id FROM quotes WHERE LOWER(text) = LOWER($1)", text).Scan(&existingID)
		if err == nil {
			// Quote with this text already exists
			c.JSON(http.StatusConflict, gin.H{"error": "This quote already exists in our database"})
			return
		} else if err != sql.ErrNoRows {
			// Some other database error occurred
			log.Println("Error checking for duplicate quote:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for duplicate quote"})
			return
		}

		// Insert quote into database (initially unapproved)
		var id int
		err = db.QueryRow(
			"INSERT INTO quotes (text, author, classification, approved, likes, submitter_name, created_at, updated_at) VALUES ($1, $2, $3, false, 0, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) RETURNING id",
			text, author, classification, submitterName,
		).Scan(&id)

		if err != nil {
			log.Println("Error inserting quote:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save quote"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "id": id, "message": "Quote submitted successfully and awaiting approval"})
	}
}

// addLikeToQuote adds 1 like to a quote's like attribute
func addLikeToQuote(quoteID int, db *sql.DB) error {
	_, err := db.Exec("UPDATE quotes SET likes = likes + 1 WHERE id = $1", quoteID)
	return err
}

// removeLikeFromQuote removes 1 like from a quote's like attribute
func removeLikeFromQuote(quoteID int, db *sql.DB) error {
	_, err := db.Exec("UPDATE quotes SET likes = likes - 1 WHERE id = $1 AND likes > 0", quoteID)
	return err
}

// scanQuotes is a helper function to scan rows into quote structs
func scanQuotes(rows *sql.Rows) ([]quote, error) {
	quotes := []quote{}

	for rows.Next() {
		var q quote
		var author sql.NullString
		if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
			log.Println(err)
			return nil, err
		}

		if author.Valid {
			q.Author = author.String
		} else {
			q.Author = ""
		}

		quotes = append(quotes, q)
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	return quotes, nil
}

// scanQuotesWithSubmitter is a helper function to scan rows into quote structs including submitter_name
func scanQuotesWithSubmitter(rows *sql.Rows) ([]quote, error) {
	quotes := []quote{}

	for rows.Next() {
		var q quote
		var author sql.NullString
		var submitterName sql.NullString
		if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes, &submitterName); err != nil {
			log.Println(err)
			return nil, err
		}

		if author.Valid {
			q.Author = author.String
		} else {
			q.Author = ""
		}

		if submitterName.Valid {
			q.SubmitterName = submitterName.String
		} else {
			q.SubmitterName = ""
		}

		quotes = append(quotes, q)
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	return quotes, nil
}

// scanQuotesWithSubmitterAndTimestamps is a helper function to scan rows into quote structs including submitter_name and timestamps
func scanQuotesWithSubmitterAndTimestamps(rows *sql.Rows) ([]quote, error) {
	quotes := []quote{}

	for rows.Next() {
		var q quote
		var author sql.NullString
		var submitterName sql.NullString
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes, &submitterName, &createdAt, &updatedAt); err != nil {
			log.Println(err)
			return nil, err
		}

		if author.Valid {
			q.Author = author.String
		} else {
			q.Author = ""
		}

		if submitterName.Valid {
			q.SubmitterName = submitterName.String
		} else {
			q.SubmitterName = ""
		}

		if createdAt.Valid {
			q.CreatedAt = &createdAt.Time
		} else {
			q.CreatedAt = nil
		}

		if updatedAt.Valid {
			q.UpdatedAt = &updatedAt.Time
		} else {
			q.UpdatedAt = nil
		}

		quotes = append(quotes, q)
	}

	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	return quotes, nil
}
