package auth

import (
	"net/http"
	"strings"

	"streamly/internal/middleware"

	"github.com/gin-gonic/gin"
)

func AuthRequired(auth *AuthService) gin.HandlerFunc {

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

		c.Set(middleware.UserIDKey, claims.UserID)

		c.Set(middleware.UserEmailKey, claims.Email)

		c.Set(middleware.IsAdminKey, claims.IsAdmin)

		c.Next()

	}

}

func AdminRequired() gin.HandlerFunc {

	return func(c *gin.Context) {

		isAdmin, ok := c.Get(middleware.IsAdminKey)

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
