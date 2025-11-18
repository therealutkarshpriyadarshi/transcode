package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/therealutkarshpriyadarshi/transcode/pkg/models"
)

const (
	AuthContextKey = "user_id"
)

var jwtSecret string

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// SetJWTSecret sets the JWT secret for the middleware
func SetJWTSecret(secret string) {
	jwtSecret = secret
}

// JWTAuth middleware validates JWT tokens
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Add user ID to context
		c.Set(AuthContextKey, claims.UserID)
		c.Next()
	}
}

// APIKeyAuth middleware validates API keys
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, apiKey string) (*models.User, error)
}

func APIKeyAuth(validator APIKeyValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		user, err := validator.ValidateAPIKey(c.Request.Context(), apiKey)
		if err != nil || user == nil || !user.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		// Add user ID to context
		c.Set(AuthContextKey, user.ID)
		c.Next()
	}
}

// OptionalAuth middleware tries both JWT and API key authentication
func OptionalAuth(validator APIKeyValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try JWT first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token, err := jwt.ParseWithClaims(parts[1], &Claims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(jwtSecret), nil
				})
				if err == nil && token.Valid {
					if claims, ok := token.Claims.(*Claims); ok {
						c.Set(AuthContextKey, claims.UserID)
						c.Next()
						return
					}
				}
			}
		}

		// Try API key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			user, err := validator.ValidateAPIKey(c.Request.Context(), apiKey)
			if err == nil && user != nil && user.IsActive {
				c.Set(AuthContextKey, user.ID)
				c.Next()
				return
			}
		}

		// No valid auth found
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Valid authentication required"})
		c.Abort()
	}
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, email string, expiresIn time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// GetUserID retrieves the user ID from the context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(AuthContextKey)
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	return userIDStr, ok
}
