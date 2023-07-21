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
)

type quote struct {
	ID             int             `json:"id"`
	Text           string          `json:"text"`
	Author         *sql.NullString `json:"author"`
	Classification string          `json:"classification"`
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
			if err := rows.Scan(&q.ID, &q.Text, &q.Author, &q.Classification); err != nil {
				log.Println(err)
				log.Fatal(err)
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
			q.Author = &author

			quotes = append(quotes, q)
		}

		if err := rows.Err(); err != nil {
			log.Println(err)
			log.Fatal(err)
		}

		c.IndentedJSON(http.StatusOK, quotes)
	})

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
			// Quote already exists, return the existing quote
			var existingQuote quote
			err := db.QueryRow("SELECT id, text, author, classification FROM quotes WHERE id=$1", existingID).Scan(&existingQuote.ID, &existingQuote.Text, &existingQuote.Author, &existingQuote.Classification)
			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve existing quote from the database."})
				return
			}
			c.IndentedJSON(http.StatusOK, existingQuote)
			return
		} else if err != sql.ErrNoRows {
			// Error occurred while querying the database
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to check if quote already exists in the database."})
			return
		}

		// Insert the new quote into the database
		var id int
		if q.Author == "" {
			q.Author = "NULL"
		} else {
			// Capitalize every word in the author field
			q.Author = strings.Title(q.Author)
		}
		if q.Classification == "" {
			q.Classification = "NULL"
		} else {
			q.Classification = strings.ToLower(q.Classification) // Convert classification to lowercase
		}
		err = db.QueryRow("INSERT INTO quotes (text, author, classification) VALUES ($1, $2, LOWER($3)) RETURNING id", q.Text, q.Author, q.Classification).Scan(&id)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to insert quote into the database."})
			return
		}
		q.ID = id

		// Return the newly created quote
		c.IndentedJSON(http.StatusCreated, q)
	})

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
