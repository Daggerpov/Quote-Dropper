package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"embed"
    "html/template"
)

type quote struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Author string `json:"author"`
	Type   string `json:"type"`
}

var quotes = []quote{
	{ID: "1", Text: "Devote the rest of your life to making progress.", Author: "Leo Tolstoy", Type: "Motivation"},
	{ID: "2", Text: "Instead of fighting the world, kill your ego.", Author: "Rumi", Type: "Self"},
	{ID: "3", Text: "A man who fears suffering is already suffering from what he fears", Author: "Montaigne", Type: "Philosophy"},
}

func getQuotes(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, quotes)
}

func quoteById(c *gin.Context) {
	id := c.Param("id")
	quote, err := getQuoteById(id)

	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Quote not found."})
		return
	}

	c.IndentedJSON(http.StatusOK, quote)
}

// func editQuote(c *gin.Context) {
// 	id, ok := c.GetQuery("id")

// 	if !ok {
// 		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Missing id query parameter."})
// 		return
// 	}

// 	quote, err := getQuoteById(id)

// 	if err != nil {
// 		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Quote not found."})
// 		return
// 	}

// 	c.IndentedJSON(http.StatusOK, quote)
// }

func getQuoteById(id string) (*quote, error) {
	for i, b := range quotes {
		if b.ID == id {
			return &quotes[i], nil
		}
	}

	return nil, errors.New("quote not found")
}

func createQuote(c *gin.Context) {
	var newQuote quote

	if err := c.BindJSON(&newQuote); err != nil {
		return
	}

	quotes = append(quotes, newQuote)
	c.IndentedJSON(http.StatusCreated, newQuote)
}

//go:embed templates/*
var resources embed.FS

var t = template.Must(template.ParseFS(resources, "templates/*"))

func main() {
	port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    	r := gin.Default()

	// GET /quotes - get all quotes
	r.GET("/quotes", getQuotes)

	// GET /quotes/:id - get a specific quote by ID
	r.GET("/quotes/:id", quoteById)

	// POST /quotes - create a new quote
	r.POST("/quotes", createQuote)

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