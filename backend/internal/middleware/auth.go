package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"streamly/internal/services"
)

const UserIDKey = "userId"
const UserEmailKey = "userEmail"
const IsAdminKey = "isAdmin"

func AuthRequired(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		claims, err := auth.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)
		c.Set(IsAdminKey, claims.IsAdmin)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, ok := c.Get(IsAdminKey)
		if !ok || !isAdmin.(bool) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	if cookie, err := c.Cookie("streamly_token"); err == nil && cookie != "" {
		return cookie
	}

	header := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}

	return ""
}