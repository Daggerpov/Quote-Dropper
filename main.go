package main

import (
	"database/sql"
	"embed"
	"strconv"
	"strings"

	// "errors"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	// for authentication for admin site
	"crypto/subtle"
	"math/rand"
)

type quote struct {
	ID             int    `json:"id"`
	Text           string `json:"text"`
	Author         string `json:"author"`
	Classification string `json:"classification"`
	Approved       bool   `json:"approved"` // New field for approval status
	Likes          int    `json:"likes"`    // New field for likes count

	// New editable fields
	EditText           string `json:"edit_text"`
	EditAuthor         string `json:"edit_author"`
	EditClassification string `json:"edit_classification"`
}

//go:embed templates/*
var resources embed.FS

var t = template.Must(template.ParseFS(resources, "templates/*"))

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Println(err)
		log.Fatal(err)
	}
	defer db.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// ADMIN STUFF

	// Define admin username and password
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	// GET /quotes - get all quotes
	r.GET("/quotes", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE approved = true")
		if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				log.Fatal(err)
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
			log.Fatal(err)
		}

		c.IndentedJSON(http.StatusOK, quotes)
	})

	// GET /quotes - get all quotes
	r.GET("/quotes/maxQuoteLength=:maxQuoteLength", func(c *gin.Context) {
		maxQuoteLengthStr := c.Param("maxQuoteLength")

		// Convert maxQuoteLength to an integer
		maxQuoteLength, err := strconv.Atoi(maxQuoteLengthStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid maxQuoteLength"})
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
		// log.Printf("maxQuoteLength value:")
		// log.Printf(maxQuoteLength)
		log.Printf("maxQuoteLength value: %d", maxQuoteLength)
		
		rows, err := db.Query(query)
		if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				log.Fatal(err)
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
			log.Fatal(err)
		}

		c.IndentedJSON(http.StatusOK, quotes)
	})

	// GET /quotes/from/:id - get quotes starting from a specific ID
	r.GET("/quotes/from/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE id >= $1 AND approved = true ORDER BY id", id)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes from the specified ID onwards"})
			return
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan quote"})
				return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while iterating rows"})
			return
		}

		c.JSON(http.StatusOK, quotes)
	})

	// GET /quotes/recent/:limit - get recent quotes with approved value of true
	r.GET("/quotes/recent/:limit", func(c *gin.Context) {
		limit := c.Param("limit")

		// Validate the limit parameter
		numLimit, err := strconv.Atoi(limit)
		if err != nil || numLimit < 1 || numLimit > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter. It should be a number between 1 and 5."})
			return
		}

		// Fetch the most recent quotes with the specified limit and approved value of true
		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE approved = true ORDER BY id DESC LIMIT $1", numLimit)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent approved quotes"})
			return
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan quote"})
				return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while iterating rows"})
			return
		}

		c.JSON(http.StatusOK, quotes)
	})

	// GET /quotes/:id - get a specific quote by ID
	r.GET("/quotes/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		var q quote
		err = db.QueryRow("SELECT id, text, author, classification, likes FROM quotes WHERE approved = TRUE AND id = $1", id).Scan(&q.ID, &q.Text, &q.Author, &q.Classification, &q.Likes)
		if err == sql.ErrNoRows {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Quote not found."})
			return
		} else if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		c.IndentedJSON(http.StatusOK, q)
	})

	// GET /quotes/randomQuote/classification=:classification - get random quote by classification
	r.GET("/quotes/randomQuote/classification=:classification", func(c *gin.Context) {
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
				c.IndentedJSON(http.StatusOK, q)
				return
			}

			// If the quote is not found or another error occurs, log it and try again
			if err != sql.ErrNoRows {
				log.Println(err)
				log.Fatal(err)
			}
		}

		// If no valid quote is found after maximum attempts, return a 404 response
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "No random quote found with the specified classification."})
	})


	// GET /quotes/classification=:classification - get quotes by classification
	r.GET("/quotes/classification=:classification", func(c *gin.Context) {
		classification := c.Param("classification")
		
		// Extract maxQuoteLength from query parameters, default to -1 (no limit)
		maxQuoteLengthParam := c.DefaultQuery("maxQuoteLength", "-1")
		maxQuoteLength, err := strconv.Atoi(maxQuoteLengthParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid maxQuoteLength"})
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
		// log.Printf("maxQuoteLength value:")
		// log.Printf(maxQuoteLength)
		log.Printf("maxQuoteLength value: %d", maxQuoteLength)


		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
			return
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan row"})
				return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Rows error"})
			return
		}

		c.IndentedJSON(http.StatusOK, quotes)
	})

	// GET /quotes/classification=:classification - get quotes by classification // TODO comemnt better
	r.GET("/quotes/classification=:classification/maxQuoteLength=:maxQuoteLength", func(c *gin.Context) {
		classification := c.Param("classification")
		maxQuoteLengthStr := c.Param("maxQuoteLength")

		// Convert maxQuoteLength to an integer
		maxQuoteLength, err := strconv.Atoi(maxQuoteLengthStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid maxQuoteLength"})
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
		// log.Printf("maxQuoteLength value:")
		// log.Printf(maxQuoteLength)
		log.Printf("maxQuoteLength value: %d", maxQuoteLength)

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
			return
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan row"})
				return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Rows error"})
			return
		}

		c.IndentedJSON(http.StatusOK, quotes)
	})


	// GET /quotes/author=:author - get quotes by author
	r.GET("/quotes/author=:author", func(c *gin.Context) {
		author := c.Param("author")

		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE author = $1 AND approved = true", author)
		if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				log.Fatal(err)
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
			log.Fatal(err)
		}

		c.IndentedJSON(http.StatusOK, quotes)
	})

	// GET /quotes/author=:author/index=:index - get a specific quote by author and index
	r.GET("/quotes/author=:author/index=:index", func(c *gin.Context) {
		author := c.Param("author")
		indexStr := c.Param("index")
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index parameter"})
			return
		}

		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE author = $1 AND approved = true ORDER BY id", author)
		if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				log.Fatal(err)
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
			log.Fatal(err)
		}

		if index < 0 || index >= len(quotes) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Index out of range"})
			return
		}

		c.IndentedJSON(http.StatusOK, quotes[index])
	})



	// GET /quoteCount?category=:category - get the number of quotes in a given category
	r.GET("/quoteCount", func(c *gin.Context) {
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
	})

	// GET /quoteLikes/:id - get the number of likes for a specific quote by ID
	r.GET("/quoteLikes/:id", func(c *gin.Context) {
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
	})

	// --------------------------------------------------------------------

	// GET METHODS ABOVE

	// POST METHODS BELOW

	// --------------------------------------------------------------------

	// POST /quotes - add a new quote
	r.POST("/quotes", func(c *gin.Context) {
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
	})

	// POST /quotes/like/:id - Increment likes for a quote by ID
	r.POST("/quotes/like/:id", func(c *gin.Context) {
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
	})

	// POST /quotes/unlike/:id - Decrement likes for a quote by ID
	r.POST("/quotes/unlike/:id", func(c *gin.Context) {
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
	})
	// --------------------------------------------------------------------

	// POST METHOD ABOVE

	// END OF PUBLIC METHODS

	// START OF ADMIN METHODS

	// --------------------------------------------------------------------

	// GET /admin - Admin page to manage unapproved quotes
	r.GET("/admin", BasicAuth(adminUsername, adminPassword), func(c *gin.Context) {
		// Query the database for unapproved quotes
		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE approved = false")
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve unapproved quotes from the database."})
			return
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to scan unapproved quotes from the database."})
				return
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
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Error occurred while retrieving unapproved quotes from the database."})
			return
		}

		// Render the admin page template with unapproved quotes
		c.HTML(http.StatusOK, "admin.html.tmpl", gin.H{"quotes": quotes})
	})

	// POST /admin/approve/:id - Approve a quote
	r.POST("/admin/approve/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		// Update the quote in the database to set approved to true
		_, err = db.Exec("UPDATE quotes SET approved = true WHERE id = $1", id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to approve the quote."})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Quote approved successfully."})
	})

	// POST /admin/dismiss/:id - Dismiss (delete) a quote
	r.POST("/admin/dismiss/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		// Delete the quote from the database
		_, err = db.Exec("DELETE FROM quotes WHERE id = $1", id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to dismiss the quote."})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Quote dismissed successfully."})
	})

	// GET /admin/search/:keyword - Search quotes by keyword
	r.GET("/admin/search/:keyword", func(c *gin.Context) {
		keyword := c.Param("keyword")

		// Execute search query in the database with a parameterized query to prevent SQL injection
		rows, err := db.Query("SELECT id, text, author, classification, likes FROM quotes WHERE text ILIKE '%' || $1 || '%' LIMIT 5", keyword)

		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to search quotes from the database."})
			return
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to scan search results from the database."})
				return
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
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Error occurred while retrieving search results from the database."})
			return
		}

		// Return the search results
		c.IndentedJSON(http.StatusOK, quotes)
	})

	// GET /admin/search/author/:author - Search quotes by author
	r.GET("/admin/search/author/:author", func(c *gin.Context) {
		author := c.Param("author")
		// Replace hyphens with spaces and convert to lowercase
		author = strings.ReplaceAll(strings.ToLower(author), "-", " ")

		// Execute search query in the database with a parameterized query to prevent SQL injection
		rows, err := db.Query("SELECT id, text, author, classification FROM quotes WHERE lower(author) LIKE '%' || $1 || '%'", author)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to search quotes from the database."})
			return
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification, &q.Likes); err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to scan search results from the database."})
				return
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
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Error occurred while retrieving search results from the database."})
			return
		}

		// Return the search results
		c.IndentedJSON(http.StatusOK, quotes)
	})

	// POST /admin/edit/:id - Edit a quote
	r.POST("/admin/edit/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid quote ID."})
			return
		}

		// Parse the request body into a new quote struct
		var editedQuote quote
		if err := c.ShouldBindJSON(&editedQuote); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid request body."})
			return
		}

		// Check if the edited quote text already exists in the database
		var existingID int
		err = db.QueryRow("SELECT id FROM quotes WHERE text = $1 AND id != $2", editedQuote.EditText, id).Scan(&existingID)
		if err == nil {
			// Edited quote text already exists, return an error
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"message": "Edited quote text already exists in the database."})
			return
		} else if err != sql.ErrNoRows {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to check if edited quote already exists in the database."})
			return
		}

		// Update the quote in the database with the edited values
		// ! not sure about following line args:
		_, err = db.Exec("UPDATE quotes SET text = $1, author = $2, classification = $3 WHERE id = $4",
			editedQuote.EditText, editedQuote.EditAuthor, editedQuote.EditClassification, id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to update the quote."})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Quote updated successfully."})
	})

	// --------------------------------------------------------------------

	// ADMIN METHODS ABOVE

	// END OF ALL METHODS
	// --------------------------------------------------------------------

	// serve static files
	r.Static("/static", "./static")

	// serve templates
	r.SetHTMLTemplate(t)

	r.GET("/", func(c *gin.Context) {
		data := map[string]string{
			"Region": os.Getenv("FLY_REGION"),
		}
		c.HTML(http.StatusOK, "index.html.tmpl", data)
	})

	log.Println("listening on", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatal(err)
	}
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

// Implement the method to add 1 like to a quote's like attribute
func addLikeToQuote(quoteID int, db *sql.DB) error {
	_, err := db.Exec("UPDATE quotes SET likes = likes + 1 WHERE id = $1", quoteID)
	if err != nil {
		return err
	}
	return nil
}

// Implement the method to remove 1 like from a quote's like attribute
func removeLikeFromQuote(quoteID int, db *sql.DB) error {
	_, err := db.Exec("UPDATE quotes SET likes = likes - 1 WHERE id = $1 AND likes > 0", quoteID)
	if err != nil {
		return err
	}
	return nil
}
