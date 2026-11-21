package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotifyNewFeedbackSendsEscapedEmail(t *testing.T) {
	var got map[string]any
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	resendURL = srv.URL
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("FEEDBACK_NOTIFY_EMAIL", "owner@example.com")

	notifyNewFeedback(7, "bug", "", "Widget <script>breaks</script>", true)

	if auth != "Bearer re_test" {
		t.Fatalf("auth header = %q", auth)
	}
	if to := got["to"].([]any)[0]; to != "owner@example.com" {
		t.Fatalf("to = %v", to)
	}
	if s := got["subject"].(string); !strings.Contains(s, "bug") || !strings.Contains(s, "Widget") {
		t.Fatalf("subject = %q", s)
	}
	h := got["html"].(string)
	if strings.Contains(h, "<script>") || !strings.Contains(h, "&lt;script&gt;") || !strings.Contains(h, "Anonymous") {
		t.Fatalf("html not escaped or missing name: %q", h)
	}
}

func TestNotifyNewFeedbackDisabledWithoutConfig(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer srv.Close()
	resendURL = srv.URL
	t.Setenv("RESEND_API_KEY", "")
	t.Setenv("FEEDBACK_NOTIFY_EMAIL", "")
	notifyNewFeedback(1, "general", "x", "y", false)
	if called {
		t.Fatal("sent an email with no config")
	}
}
