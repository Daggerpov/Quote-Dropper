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
	// Get all quotes
	r.GET("/quotes", handleGetAllQuotes(db))

	// Get quotes with max length constraint
	r.GET("/quotes/maxQuoteLength=:maxQuoteLength", handleGetQuotesWithMaxLength(db))

	// Get quotes starting from a specific ID
	r.GET("/quotes/from/:id", handleGetQuotesFromID(db))

	// Get recent quotes
	r.GET("/quotes/recent/:limit", handleGetRecentQuotes(db))

	// Get top quotes (most liked)
	r.GET("/quotes/top", handleGetTopQuotes(db))

	// Get a specific quote by ID
	r.GET("/quotes/:id", handleGetQuoteByID(db))

	// Get random quote by classification
	r.GET("/quotes/randomQuote/classification=:classification", handleGetRandomQuoteByClassification(db))

	// Get quotes by classification
	r.GET("/quotes/classification=:classification", handleGetQuotesByClassification(db))

	// Get quotes by classification with max length
	r.GET("/quotes/classification=:classification/maxQuoteLength=:maxQuoteLength", handleGetQuotesByClassificationWithMaxLength(db))

	// Get quotes by author
	r.GET("/quotes/author=:author", handleGetQuotesByAuthor(db))

	// Get a specific quote by author and index
	r.GET("/quotes/author=:author/index=:index", handleGetQuoteByAuthorAndIndex(db))

	// Get quote count by category
	r.GET("/quoteCount", handleGetQuoteCount(db))

	// Get likes for a specific quote
	r.GET("/quoteLikes/:id", handleGetQuoteLikes(db))

	// Add a new quote
	r.POST("/quotes", handleAddQuote(db))

	// Increment likes for a quote
	r.POST("/quotes/like/:id", handleLikeQuote(db))

	// Decrement likes for a quote
	r.POST("/quotes/unlike/:id", handleUnlikeQuote(db))

	// Render quote submission page
	r.GET("/submit-quote", func(c *gin.Context) {
		c.HTML(http.StatusOK, "submit-quote.html.tmpl", gin.H{})
	})

	// Handle quote submission
	r.POST("/submit-quote", handleSubmitQuote(db))
}

// handleGetAllQuotes returns a handler for getting all approved quotes
func handleGetAllQuotes(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE approved = true")
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

		if isBrowserRequest(c) {
			stats := &QuoteStats{Count: len(quotes)}
			renderQuotesHTML(c, quotes, "All Quotes", "Browse all approved quotes in the database", stats)
		} else {
			c.IndentedJSON(http.StatusOK, quotes)
		}
	}
}

// handleGetQuotesWithMaxLength returns a handler for getting quotes with max length constraint
func handleGetQuotesWithMaxLength(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		maxQuoteLengthStr := c.Param("maxQuoteLength")

		// Convert maxQuoteLength to an integer
		maxQuoteLength, err := strconv.Atoi(maxQuoteLengthStr)
		if err != nil {
			if isBrowserRequest(c) {
				c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
					Title:       "Invalid Request",
					Description: "Invalid maximum quote length specified",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid maxQuoteLength"})
			}
			return
		}

		query := "SELECT id, text, author, classification, likes FROM quotes WHERE approved = true"

		// Append additional condition if maxQuoteLength is valid
		if maxQuoteLength >= 0 {
			query += " AND LENGTH(text) <= "
			query += maxQuoteLengthStr
		}

		// Log the final query for debugging
		log.Printf("Executing query: %s with no args. (for all category)", query)
		log.Printf("maxQuoteLength value: %d", maxQuoteLength)

		rows, err := db.Query(query)
		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Database query failed",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
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
			stats := &QuoteStats{Count: len(quotes), MaxLength: maxQuoteLength}
			title := "Short Quotes"
			description := "Quotes with maximum " + maxQuoteLengthStr + " characters"
			renderQuotesHTML(c, quotes, title, description, stats)
		} else {
			c.IndentedJSON(http.StatusOK, quotes)
		}
	}
}

// handleGetQuotesFromID returns a handler for getting quotes starting from a specific ID
func handleGetQuotesFromID(db *sql.DB) gin.HandlerFunc {
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

		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE id >= $1 AND approved = true ORDER BY id", id)
		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Failed to fetch quotes from the specified ID onwards",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes from the specified ID onwards"})
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
			title := "Quotes from ID " + idStr
			description := "All quotes starting from ID " + idStr
			renderQuotesHTML(c, quotes, title, description, stats)
		} else {
			c.JSON(http.StatusOK, quotes)
		}
	}
}

// handleGetRecentQuotes returns a handler for getting recent quotes
func handleGetRecentQuotes(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := c.Param("limit")

		// Validate the limit parameter
		numLimit, err := strconv.Atoi(limit)
		if err != nil || numLimit < 1 || numLimit > 5 {
			if isBrowserRequest(c) {
				c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
					Title:       "Invalid Request",
					Description: "Invalid limit parameter. It should be a number between 1 and 5.",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter. It should be a number between 1 and 5."})
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
			description := "The " + limit + " most recently added quotes"
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

// handleGetRandomQuoteByClassification returns a handler for getting a random quote by classification
func handleGetRandomQuoteByClassification(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		classification := c.Param("classification")

		// Define the range of IDs to search
		const maxAttempts = 10
		const minID = 1
		const maxID = 500 // Update this value as per your maximum ID

		var q quote
		var err error
		var author sql.NullString

		for attempts := 0; attempts < maxAttempts; attempts++ {
			// Generate a random ID within the range
			randID := rand.Intn(maxID-minID+1) + minID

			// Try to fetch the quote with the generated ID
			err = db.QueryRow(`
				SELECT id, text, author, classification, likes 
				FROM quotes 
				WHERE id = $1 AND classification = $2 AND approved = TRUE`, randID, classification).Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes)

			if err == nil {
				if author.Valid {
					q.Author = author.String
				} else {
					q.Author = ""
				}
				// If the quote is found, return it
				if isBrowserRequest(c) {
					stats := &QuoteStats{Count: 1, Classification: classification}
					title := "Random " + cases.Title(language.English).String(classification) + " Quote"
					description := "A randomly selected quote from the " + classification + " category"
					renderQuotesHTML(c, []quote{q}, title, description, stats)
				} else {
					c.IndentedJSON(http.StatusOK, q)
				}
				return
			}

			// If the quote is not found or another error occurs, log it and try again
			if err != sql.ErrNoRows {
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
		}

		// If no valid quote is found after maximum attempts, return a 404 response
		if isBrowserRequest(c) {
			c.HTML(http.StatusNotFound, "quotes.html.tmpl", QuotePageData{
				Title:       "No Quote Found",
				Description: "No random quote found with the specified classification: " + classification,
				Quotes:      []quote{},
			})
		} else {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "No random quote found with the specified classification."})
		}
	}
}

// handleGetQuotesByClassification returns a handler for getting quotes by classification
func handleGetQuotesByClassification(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		classification := c.Param("classification")

		// Extract maxQuoteLength from query parameters, default to -1 (no limit)
		maxQuoteLengthParam := c.DefaultQuery("maxQuoteLength", "-1")
		maxQuoteLength, err := strconv.Atoi(maxQuoteLengthParam)
		if err != nil {
			if isBrowserRequest(c) {
				c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
					Title:       "Invalid Request",
					Description: "Invalid maxQuoteLength parameter",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid maxQuoteLength"})
			}
			return
		}

		// Prepare the SQL query
		query := "SELECT id, text, author, classification, likes FROM quotes WHERE classification = $1 AND approved = true"
		args := []interface{}{classification}

		// Append additional condition if maxQuoteLength is valid
		if maxQuoteLength >= 0 {
			query += " AND LENGTH(text) <= $2"
			args = append(args, maxQuoteLength)
		}

		// Log the final query for debugging
		log.Printf("Executing query: %s with args: %v", query, args)
		log.Printf("maxQuoteLength value: %d", maxQuoteLength)

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Database query failed",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
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
			stats := &QuoteStats{Count: len(quotes), Classification: classification}
			if maxQuoteLength >= 0 {
				stats.MaxLength = maxQuoteLength
			}
			title := cases.Title(language.English).String(classification) + " Quotes"
			description := "All quotes in the " + classification + " category"
			if maxQuoteLength >= 0 {
				description += " with maximum " + maxQuoteLengthParam + " characters"
			}
			renderQuotesHTML(c, quotes, title, description, stats)
		} else {
			c.IndentedJSON(http.StatusOK, quotes)
		}
	}
}

// handleGetQuotesByClassificationWithMaxLength returns a handler for getting quotes by classification with max length
func handleGetQuotesByClassificationWithMaxLength(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		classification := c.Param("classification")
		maxQuoteLengthStr := c.Param("maxQuoteLength")

		// Convert maxQuoteLength to an integer
		maxQuoteLength, err := strconv.Atoi(maxQuoteLengthStr)
		if err != nil {
			if isBrowserRequest(c) {
				c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
					Title:       "Invalid Request",
					Description: "Invalid maxQuoteLength parameter",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid maxQuoteLength"})
			}
			return
		}

		query := "SELECT id, text, author, classification, likes FROM quotes WHERE classification = $1 AND approved = true"

		// Append additional condition if maxQuoteLength is valid
		if maxQuoteLength >= 0 {
			query += " AND LENGTH(text) <= $2"
		}

		// Log the final query for debugging
		log.Printf("Executing query: %s with args: [%s, %d]", query, classification, maxQuoteLength)

		var rows *sql.Rows
		if maxQuoteLength >= 0 {
			rows, err = db.Query(query, classification, maxQuoteLength)
		} else {
			rows, err = db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE classification = $1 AND approved = true", classification)
		}

		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Database query failed",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
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
			stats := &QuoteStats{Count: len(quotes), MaxLength: maxQuoteLength, Classification: classification}
			title := "Short " + cases.Title(language.English).String(classification) + " Quotes"
			description := cases.Title(language.English).String(classification) + " quotes with maximum " + maxQuoteLengthStr + " characters"
			renderQuotesHTML(c, quotes, title, description, stats)
		} else {
			c.IndentedJSON(http.StatusOK, quotes)
		}
	}
}

// handleGetQuotesByAuthor returns a handler for getting quotes by author
func handleGetQuotesByAuthor(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		author := c.Param("author")

		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE author = $1 AND approved = true", author)
		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Failed to fetch quotes by author",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes by author"})
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
			title := "Quotes by " + author
			description := "All quotes attributed to " + author
			renderQuotesHTML(c, quotes, title, description, stats)
		} else {
			c.JSON(http.StatusOK, quotes)
		}
	}
}

// handleGetQuoteByAuthorAndIndex returns a handler for getting a specific quote by author and index
func handleGetQuoteByAuthorAndIndex(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		author := c.Param("author")
		indexStr := c.Param("index")
		index, err := strconv.Atoi(indexStr)
		if err != nil {
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

		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE author = $1 AND approved = true ORDER BY id", author)
		if err != nil {
			log.Println(err)
			if isBrowserRequest(c) {
				c.HTML(http.StatusInternalServerError, "quotes.html.tmpl", QuotePageData{
					Title:       "Error",
					Description: "Database query failed",
					Quotes:      []quote{},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
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

		if index < 0 || index >= len(quotes) {
			if isBrowserRequest(c) {
				c.HTML(http.StatusBadRequest, "quotes.html.tmpl", QuotePageData{
					Title:       "Index Out of Range",
					Description: "The specified index is out of range for quotes by " + author,
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
	}
}

// handleGetQuoteCount returns a handler for getting the number of quotes in a given category
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

// handleGetQuoteLikes returns a handler for getting the number of likes for a specific quote by ID
func handleGetQuoteLikes(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		var likes int
		err = db.QueryRow("SELECT likes FROM quotes WHERE id = $1 AND approved = true", id).Scan(&likes)
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Quote not found."})
			return
		} else if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve likes count from the database."})
			return
		}

		c.IndentedJSON(http.StatusOK, gin.H{"id": id, "likes": likes})
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
		err = db.QueryRow("INSERT INTO quotes (text, author, classification, approved, likes) VALUES ($1, $2, LOWER($3), $4, $5) RETURNING id", q.Text, q.Author, q.Classification, q.Approved, q.Likes).Scan(&id)
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
			"INSERT INTO quotes (text, author, classification, approved, likes) VALUES ($1, $2, $3, false, 0) RETURNING id",
			text, author, classification,
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

// handleGetTopQuotes returns a handler for getting the top (most liked) quotes
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
