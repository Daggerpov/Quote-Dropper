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

		// Add a period if there isn't any yet in the 'Text' field
		if q.Text[len(q.Text)-1] != '.' {
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

		// Update the approved status of the quote in the database
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
