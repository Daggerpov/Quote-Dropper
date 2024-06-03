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
		rows, err := db.Query("SELECT id, text, author, classification FROM quotes")
		if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification); err != nil {
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

		rows, err := db.Query("SELECT id, text, author, classification FROM quotes WHERE id >= $1 ORDER BY id", id)
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
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification); err != nil {
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
		rows, err := db.Query("SELECT id, text, author, classification FROM quotes WHERE approved = true ORDER BY id DESC LIMIT $1", numLimit)
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
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification); err != nil {
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
		err = db.QueryRow("SELECT id, text, author, classification FROM quotes WHERE id = $1", id).Scan(&q.ID, &q.Text, &q.Author, &q.Classification)
		if err == sql.ErrNoRows {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Quote not found."})
			return
		} else if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		c.IndentedJSON(http.StatusOK, q)
	})

	// GET /quotes/:classification - get quotes by classification
	r.GET("/quotes/classification=:classification", func(c *gin.Context) {
		classification := c.Param("classification")

		rows, err := db.Query("SELECT id, text, author, classification FROM quotes WHERE classification = $1", classification)
		if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		defer rows.Close()

		quotes := []quote{}

		for rows.Next() {
			var q quote
			var author sql.NullString
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification); err != nil {
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

	// GET /quoteCount?category=:category - get the number of quotes in a given category
	r.GET("/quoteCount", func(c *gin.Context) {
		category := c.Query("category")
		if category == "" || category == "all" {
			// If category is not specified, retrieve the total count of all quotes
			var totalCount int
			err := db.QueryRow("SELECT COUNT(*) FROM quotes").Scan(&totalCount)
			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve total quote count from the database."})
				return
			}

			c.IndentedJSON(http.StatusOK, gin.H{"category": "all", "count": totalCount})
			return
		}

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM quotes WHERE classification = $1", strings.ToLower(category)).Scan(&count)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve quote count from the database."})
			return
		}

		c.IndentedJSON(http.StatusOK, gin.H{"category": category, "count": count})
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
		if q.Text[len(q.Text)-1] != '.' && q.Text[len(q.Text)-1] != '?' {
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
		err = db.QueryRow("INSERT INTO quotes (text, author, classification, approved) VALUES ($1, $2, LOWER($3), $4) RETURNING id", q.Text, q.Author, q.Classification, q.Approved).Scan(&id)
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
		err = db.QueryRow("SELECT id, text, author, classification, likes FROM quotes WHERE id = $1", id).Scan(&updatedQuote.ID, &updatedQuote.Text, &updatedQuote.Author, &updatedQuote.Classification, &updatedQuote.Likes)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch updated quote."})
			return
		}

		// Return the updated quote with incremented like count
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
		rows, err := db.Query("SELECT id, text, author, classification FROM quotes WHERE approved = false")
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
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification); err != nil {
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
		rows, err := db.Query("SELECT id, text, author, classification, Likes FROM quotes WHERE text ILIKE '%' || $1 || '%' LIMIT 5", keyword)
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
			if err := rows.Scan(&q.ID, &q.Text, &author, &q.Classification); err != nil {
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
