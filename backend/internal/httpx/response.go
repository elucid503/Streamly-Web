package httpx

import (
	"errors"
	"net/http"
	"strings"

	"streamly/internal/apperror"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func WriteError(c *gin.Context, status int, message string) {

	c.JSON(status, gin.H{"error": message})

}

func HandleError(c *gin.Context, err error) {

	switch {

	case errors.Is(err, apperror.ErrInvalidCredentials):

		WriteError(c, http.StatusUnauthorized, "invalid email or password")

	case errors.Is(err, apperror.ErrEmailTaken):

		WriteError(c, http.StatusConflict, "email already registered")

	case errors.Is(err, apperror.ErrInvalidAccessCode):

		WriteError(c, http.StatusForbidden, "invalid or expired access code")

	case errors.Is(err, apperror.ErrInvalidFavorite):

		WriteError(c, http.StatusBadRequest, "invalid favorite")

	case errors.Is(err, apperror.ErrAccessCodeExhausted):

		WriteError(c, http.StatusForbidden, "access code has reached its usage limit")

	case errors.Is(err, mongo.ErrNoDocuments):

		WriteError(c, http.StatusNotFound, "not found")

	default:

		if isUpstreamUnavailable(err) {

			WriteError(c, http.StatusServiceUnavailable, "upstream temporarily unavailable")
			return

		}

		WriteError(c, http.StatusInternalServerError, err.Error())

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
		strings.Contains(msg, "febbox:") ||
		strings.Contains(msg, "no providers") ||
		strings.Contains(msg, "no stream available") ||
		strings.Contains(msg, "live/source")

}

func JSONSlice[T any](items []T) []T {

	if items == nil {

		return []T{}

	}

	return items

}

func BaseURL(c *gin.Context) string {

	scheme := "http"

	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {

		scheme = "https"

	}

	return scheme + "://" + c.Request.Host

}
