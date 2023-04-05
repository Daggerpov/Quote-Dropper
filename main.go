package main

import (
	"database/sql"
	"embed"
	// "errors"
	"github.com/gin-gonic/gin"
	"html/template"
	"log"
	"net/http"
	"os"
	_ "github.com/lib/pq"
)

type quote struct {
	ID     			string `json:"id"`
	Text   			string `json:"text"`
	Author 			string `json:"author"`
	Classification  string `json:"classification"`
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

	quotes := []quote{}

	// GET /quotes - get all quotes
	r.GET("/quotes", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, text, author, classification FROM quotes")
		if err != nil {
			log.Println(err)
			log.Fatal(err)
		}
		defer rows.Close()

		quotes = []quote{}
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
        id := c.Param("id")
        var q quote
        err := db.QueryRow("SELECT id, text, author, classification FROM quotes WHERE id = $1", id).Scan(&q.ID, &q.Text, &q.Author, &q.Classification)
        if err == sql.ErrNoRows {
            c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Quote not found."})
            return
        } else if err != nil {
			log.Println(err)
            log.Fatal(err)
        }
        c.IndentedJSON(http.StatusOK, q)
    })

    // POST /quotes - create a new quote
    r.POST("/quotes", func(c *gin.Context) {
        var newQuote quote
        if err := c.BindJSON(&newQuote); err != nil {
			log.Println(err)
            log.Fatal(err)
        }
        stmt, err := db.Prepare("INSERT INTO quotes (id, text, author, classification) VALUES ($1, $2, $3, $4)")
        if err != nil {
			log.Println(err)
            log.Fatal(err)
        }
        _, err = stmt.Exec(newQuote.ID, newQuote.Text, newQuote.Author, newQuote.Classification)
        if err != nil {
			log.Println(err)
            log.Fatal(err)
        }
        quotes = append(quotes, newQuote)
        c.IndentedJSON(http.StatusCreated, newQuote)
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
	log.Fatal(http.ListenAndServe(":"+port, r))
}
