package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for API requests
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps int, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    burst,
	}
}

// getLimiter returns a rate limiter for a specific key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	limiter, exists = rl.limiters[key]
	if exists {
		return limiter
	}

	limiter = rate.NewLimiter(rl.rate, rl.burst)
	rl.limiters[key] = limiter

	return limiter
}

// Cleanup removes old limiters
func (rl *RateLimiter) Cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// In production, track last access time and remove inactive limiters
		// For now, we'll keep all limiters (simple approach)
		rl.mu.Unlock()
	}
}

// RateLimit middleware limits requests per IP or user
func RateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get user ID first
		userID, exists := c.Get(AuthContextKey)
		var key string

		if exists {
			key = fmt.Sprintf("user:%s", userID)
		} else {
			// Fall back to IP address
			key = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// QuotaValidator interface for checking user quotas
type QuotaValidator interface {
	CheckQuota(ctx context.Context, userID string) (bool, error)
	IncrementQuota(ctx context.Context, userID string) error
}

// QuotaLimit middleware enforces user quotas
func QuotaLimit(validator QuotaValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			// No user ID in context, skip quota check
			c.Next()
			return
		}

		hasQuota, err := validator.CheckQuota(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check quota"})
			c.Abort()
			return
		}

		if !hasQuota {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Quota exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()

		// Increment quota after successful request
		if c.Writer.Status() < 400 {
			_ = validator.IncrementQuota(c.Request.Context(), userID)
		}
	}
}
