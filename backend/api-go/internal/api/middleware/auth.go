package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"api-go/internal/config"
	"api-go/internal/database"
	"api-go/internal/models"
	"api-go/internal/api/authutils"

	"github.com/gin-gonic/gin"
)

// RequireAuth is a middleware to ensure the request is authenticated via JWT
func RequireAuth(cfg *config.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Not authenticated"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		subject, err := authutils.DecodeAccessToken(tokenString, cfg.JWTSecret)
		if err != nil || subject == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid or expired token"})
			return
		}

		userID, err := strconv.Atoi(subject)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token payload"})
			return
		}

		var user models.User
		if err := database.DB.Preload("Domain").First(&user, userID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "User not found or inactive"})
			return
		}

		if !user.IsActive {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "Account is disabled"})
			return
		}

		// Set the user in the context
		c.Set("user", &user)
		c.Next()
	}
}

// RequireAdmin is a middleware to ensure the authenticated user is an admin
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userVal, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Not authenticated"})
			return
		}

		user := userVal.(*models.User)
		if !user.IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "Admin access required"})
			return
		}

		c.Next()
	}
}

// GetCurrentUser is a helper to get the user from context
func GetCurrentUser(c *gin.Context) *models.User {
	val, exists := c.Get("user")
	if !exists {
		return nil
	}
	return val.(*models.User)
}
