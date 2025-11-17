package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	userID := "test-user-id"
	email := "test@example.com"

	token, err := GenerateToken(userID, email, 1*time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "Missing authorization header",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token format",
			token:          "InvalidToken",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			c.Request = req

			JWTAuth()(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestJWTAuthWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Generate a valid token
	userID := "test-user-id"
	email := "test@example.com"
	token, err := GenerateToken(userID, email, 1*time.Hour)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	c.Request = req

	// Create a handler that checks if user ID is set
	handler := func(c *gin.Context) {
		extractedUserID, exists := GetUserID(c)
		assert.True(t, exists)
		assert.Equal(t, userID, extractedUserID)
		c.Status(http.StatusOK)
	}

	// Execute middleware and handler
	JWTAuth()(c)
	if !c.IsAborted() {
		handler(c)
	}

	assert.Equal(t, http.StatusOK, w.Code)
}
