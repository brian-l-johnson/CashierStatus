package middleware

import (
	"net/http"
	"strings"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthorizeBearer validates Bearer token authentication for API clients
// This allows headless clients (like merch-app) to authenticate without sessions
// Requires both Authorization: Bearer <token> AND X-API-Key-ID: <key-id> headers
// The X-API-Key-ID header enables O(1) database lookup instead of iterating all keys
func AuthorizeBearer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "missing Authorization header"})
			c.Abort()
			return
		}

		// Check for Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "invalid Authorization header format, expected 'Bearer <token>'"})
			c.Abort()
			return
		}

		plaintextKey := parts[1]

		// Require X-API-Key-ID header for optimized lookup
		keyID := c.GetHeader("X-API-Key-ID")
		if keyID == "" {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "missing X-API-Key-ID header"})
			c.Abort()
			return
		}

		// Direct lookup by key ID (avoids expensive iteration over all keys)
		var apiKey models.APIKey
		if err := db.Where("key_id = ? AND active = ?", keyID, true).First(&apiKey).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "invalid API key"})
			} else {
				c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "database error"})
			}
			c.Abort()
			return
		}

		// Validate the key
		if apiKey.ValidateKey(plaintextKey) {
			// Valid key found, set context and continue
			c.Set("api_key_id", apiKey.KeyID)
			c.Set("api_key_name", apiKey.Name)
			c.Set("api_key_purpose", apiKey.Purpose)
			c.Next()
			return
		}

		// Key ID matched but hash didn't - invalid key
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "invalid API key"})
		c.Abort()
	}
}

// AuthorizeEither accepts either session-based auth OR Bearer token auth
// This is useful for endpoints that need to work for both web UI and API clients
func AuthorizeEither(db *gorm.DB, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if Authorization header is present (Bearer token)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// Use Bearer token authentication
			AuthorizeBearer(db)(c)
			return
		}

		// Fall back to session authentication
		Authorize(role)(c)
	}
}
