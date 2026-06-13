package handlers

import (
	"errors"
	"net/http"
	"strings"

	"streamly/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		writeError(c, http.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, services.ErrEmailTaken):
		writeError(c, http.StatusConflict, "email already registered")
	case errors.Is(err, services.ErrInvalidAccessCode):
		writeError(c, http.StatusForbidden, "invalid or expired access code")
	case errors.Is(err, services.ErrAccessCodeExhausted):
		writeError(c, http.StatusForbidden, "access code has reached its usage limit")
	case errors.Is(err, mongo.ErrNoDocuments):
		writeError(c, http.StatusNotFound, "not found")
	default:
		if isUpstreamUnavailable(err) {
			writeError(c, http.StatusServiceUnavailable, "upstream temporarily unavailable")
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
	}
}

func isUpstreamUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "1015") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "525") ||
		strings.Contains(msg, "febbox:")
}

func jsonSlice[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

func baseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}
