package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var resendURL = "https://api.resend.com/emails"

// notifyNewFeedback emails FEEDBACK_NOTIFY_EMAIL about a new submission through Resend's HTTPS API
// (Railway blocks outbound SMTP). It does nothing unless RESEND_API_KEY and FEEDBACK_NOTIFY_EMAIL are set.
// Call it in a goroutine so a slow or failed send never delays the user.
func notifyNewFeedback(id int, feedbackType, name, content string, hasImage bool) {
	apiKey := os.Getenv("RESEND_API_KEY")
	to := os.Getenv("FEEDBACK_NOTIFY_EMAIL")
	if apiKey == "" || to == "" {
		return
	}
	from := os.Getenv("RESEND_FROM")
	if from == "" {
		from = "Quote Droplet <noreply@getspawn.com>"
	}
	if name == "" {
		name = "Anonymous"
	}

	preview := strings.Join(strings.Fields(content), " ")
	if r := []rune(preview); len(r) > 60 {
		preview = string(r[:60]) + "…"
	}
	body := fmt.Sprintf(
		"<h2>New Quote Droplet feedback: %s</h2><p><b>From:</b> %s</p><p style=\"white-space:pre-wrap\">%s</p>",
		html.EscapeString(feedbackType), html.EscapeString(name), html.EscapeString(content))
	if hasImage {
		body += "<p>An image was attached; see the admin page.</p>"
	}
	body += fmt.Sprintf("<p style=\"color:#888\">Feedback ID %d</p>", id)

	payload, _ := json.Marshal(map[string]any{
		"from":    from,
		"to":      []string{to},
		"subject": fmt.Sprintf("Quote Droplet feedback (%s): %s", feedbackType, preview),
		"html":    body,
	})
	req, err := http.NewRequest(http.MethodPost, resendURL, bytes.NewReader(payload))
	if err != nil {
		log.Println("Feedback email failed:", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		log.Println("Feedback email failed:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		log.Println("Feedback email failed: HTTP", resp.StatusCode)
	}
}
