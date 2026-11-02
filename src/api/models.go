package main

import (
	"time"
)

// Quote represents a quote in the database
type quote struct {
	ID             int        `json:"id"`
	Text           string     `json:"text"`
	Author         string     `json:"author"`
	Classification string     `json:"classification"`
	Approved       bool       `json:"approved"`               // New field for approval status
	Likes          int        `json:"likes"`                  // New field for likes count
	SubmitterName  string     `json:"submitter_name"`         // Name/alias of the person who submitted the quote
	SubmitterID    string     `json:"submitter_id,omitempty"` // Anonymous device ID used to look up a user's own submissions
	CreatedAt      *time.Time `json:"created_at"`             // Timestamp when quote was submitted (nullable)
	UpdatedAt      *time.Time `json:"updated_at"`             // Timestamp when quote was last updated (nullable)
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`  // Timestamp when quote was approved (nullable)

	// New editable fields
	EditText           string `json:"edit_text"`
	EditAuthor         string `json:"edit_author"`
	EditClassification string `json:"edit_classification"`
}

// submission represents a quote as seen by the user who submitted it
type submission struct {
	ID             int        `json:"id"`
	Text           string     `json:"text"`
	Author         string     `json:"author"`
	Classification string     `json:"classification"`
	Likes          int        `json:"likes"`
	SubmitterName  string     `json:"submitter_name"`
	Status         string     `json:"status"` // pending, approved, rejected
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
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
