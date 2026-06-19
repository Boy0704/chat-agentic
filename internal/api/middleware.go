package api

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		if token != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// BodySizeLimit rejects requests whose body exceeds maxBytes.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// SlogLogger is a structured request logger middleware using slog.
func SlogLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path += "?" + c.Request.URL.RawQuery
		}

		c.Next()

		args := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"size", c.Writer.Size(),
		}
		if len(c.Errors) > 0 {
			args = append(args, "errors", c.Errors.String())
		}

		if c.Writer.Status() >= 500 {
			logger.Error("request", args...)
		} else {
			logger.Info("request", args...)
		}
	}
}

// ipRateLimiter is a sliding-window per-IP rate limiter (no external dependencies).
type ipRateLimiter struct {
	mu        sync.Mutex
	buckets   map[string][]time.Time
	limit     int
	window    time.Duration
	lastClean time.Time
}

func newIPRateLimiter(requestsPerMinute int) *ipRateLimiter {
	return &ipRateLimiter{
		buckets:   make(map[string][]time.Time),
		limit:     requestsPerMinute,
		window:    time.Minute,
		lastClean: time.Now(),
	}
}

func (r *ipRateLimiter) Allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	// Periodic cleanup every 5 minutes to prevent unbounded map growth
	if now.Sub(r.lastClean) > 5*time.Minute {
		for k, v := range r.buckets {
			if len(v) == 0 || v[len(v)-1].Before(cutoff) {
				delete(r.buckets, k)
			}
		}
		r.lastClean = now
	}

	times := r.buckets[ip]

	// Drop expired entries
	start := 0
	for start < len(times) && times[start].Before(cutoff) {
		start++
	}
	times = times[start:]

	if len(times) >= r.limit {
		r.buckets[ip] = times
		return false
	}

	r.buckets[ip] = append(times, now)
	return true
}

// RateLimitMiddleware returns 429 when the per-IP request rate exceeds the limit.
func RateLimitMiddleware(rl *ipRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded — please slow down",
			})
			return
		}
		c.Next()
	}
}
