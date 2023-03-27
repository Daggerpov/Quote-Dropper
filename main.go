package main

import (
	"net/http"

	"errors"

	"github.com/gin-gonic/gin"
)

type quote struct {
	ID       	string `json:"id"`
	Text    	string `json:"text"`
	Author   	string `json:"author"`
	Type 		string `json:"type"`
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

func main() {
	router := gin.Default()
	router.GET("/quotes", getQuotes)
	router.GET("/quotes/:id", quoteById)
	router.POST("/quotes", createQuote)
	// router.PATCH("/edit", editQuote) TODO - Implement editing quotes.
	router.Run("localhost:8080")
}