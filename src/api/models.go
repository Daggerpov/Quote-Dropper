package main

import (
	"time"
)

// Quote represents a quote in the database
type quote struct {
	ID             int    `json:"id"`
	Text           string `json:"text"`
	Author         string `json:"author"`
	Classification string `json:"classification"`
	Approved       bool   `json:"approved"`       // New field for approval status
	Likes          int    `json:"likes"`          // New field for likes count
	SubmitterName  string `json:"submitter_name"` // New field for submitter's name

	// New editable fields
	EditText           string `json:"edit_text"`
	EditAuthor         string `json:"edit_author"`
	EditClassification string `json:"edit_classification"`
}

// Feedback represents user feedback in the database
type feedback struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`       // general, bug, feature
	Name      string    `json:"name"`       // Optional name/alias
	Content   string    `json:"content"`    // Feedback content
	ImagePath string    `json:"image_path"` // Path to uploaded image, if any
	CreatedAt time.Time `json:"created_at"` // Timestamp
}
