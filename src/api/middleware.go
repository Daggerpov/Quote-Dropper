package main

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a simple rate limiter implementation
type RateLimiter struct {
	requests     map[string][]time.Time
	mu           sync.Mutex
	maxRequests  int           // Maximum number of requests allowed in the window
	windowPeriod time.Duration // Time window for rate limiting
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(maxRequests int, windowPeriod time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:     make(map[string][]time.Time),
		maxRequests:  maxRequests,
		windowPeriod: windowPeriod,
	}
}

// Allow checks if a request from a given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean up old requests
	if _, exists := rl.requests[ip]; exists {
		var validRequests []time.Time
		for _, t := range rl.requests[ip] {
			if now.Sub(t) <= rl.windowPeriod {
				validRequests = append(validRequests, t)
			}
		}
		rl.requests[ip] = validRequests
	}

	// Check if the IP has reached the limit
	if len(rl.requests[ip]) >= rl.maxRequests {
		return false
	}

	// Add the current request
	rl.requests[ip] = append(rl.requests[ip], now)
	return true
}

// RateLimitMiddleware is a Gin middleware that implements rate limiting
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			ip = c.Request.RemoteAddr
		}

		// Check for X-Forwarded-For header (common in proxy setups)
		if forwardedFor := c.Request.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			// Use the first IP in the list
			ips := strings.Split(forwardedFor, ",")
			ip = strings.TrimSpace(ips[0])
		}

		// Allow all localhost requests (for development)
		if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
			c.Next()
			return
		}

		// Check if the request is allowed
		if !rl.Allow(ip) {
			c.AbortWithStatusJSON(429, gin.H{"error": "Rate limit exceeded"})
			return
		}

		c.Next()
	}
}
