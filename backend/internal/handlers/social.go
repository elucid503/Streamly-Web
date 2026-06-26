package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"streamly/internal/middleware"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type SocialHandler struct {

	social *services.SocialService
}

func NewSocialHandler(social *services.SocialService) *SocialHandler {

	return &SocialHandler{social: social}

}

func (h *SocialHandler) GetMyProfile(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	profile, err := h.social.GetMyProfile(c.Request.Context(), userID)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, profile)

}

func (h *SocialHandler) UpdateProfile(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	var input services.ProfileUpdate

	if err := c.ShouldBindJSON(&input); err != nil {

		writeError(c, http.StatusBadRequest, "invalid request")
		return

	}

	profile, err := h.social.UpdateProfile(c.Request.Context(), userID, input)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, profile)

}

func (h *SocialHandler) GetPublicProfile(c *gin.Context) {

	viewerID := c.GetString(middleware.UserIDKey)
	targetID := c.Param("id")

	profile, err := h.social.GetPublicProfile(c.Request.Context(), viewerID, targetID)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, profile)

}

func (h *SocialHandler) SearchUsers(c *gin.Context) {

	viewerID := c.GetString(middleware.UserIDKey)
	query := c.Query("q")

	users, err := h.social.SearchUsers(c.Request.Context(), viewerID, query)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, users)

}

func (h *SocialHandler) ListFriends(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	friends, err := h.social.ListFriends(c.Request.Context(), userID)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, friends)

}

func (h *SocialHandler) ListRequests(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	requests, err := h.social.ListRequests(c.Request.Context(), userID)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, requests)

}

func (h *SocialHandler) SendRequest(c *gin.Context) {

	fromID := c.GetString(middleware.UserIDKey)

	var body struct {
		ToID string `json:"toId"`
	}

	if err := c.ShouldBindJSON(&body); err != nil || body.ToID == "" {

		writeError(c, http.StatusBadRequest, "toId required")
		return

	}

	if err := h.social.SendRequest(c.Request.Context(), fromID, body.ToID); err != nil {

		handleServiceError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}

func (h *SocialHandler) AcceptRequest(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)
	requestID := c.Param("id")

	if err := h.social.AcceptRequest(c.Request.Context(), requestID, userID); err != nil {

		handleServiceError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}

func (h *SocialHandler) DeleteRequest(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)
	requestID := c.Param("id")

	if err := h.social.DeleteRequest(c.Request.Context(), requestID, userID); err != nil {

		handleServiceError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}

func (h *SocialHandler) RemoveFriend(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)
	friendID := c.Param("id")

	if err := h.social.RemoveFriend(c.Request.Context(), userID, friendID); err != nil {

		handleServiceError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}

func (h *SocialHandler) SSEEvents(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := h.social.Hub().Subscribe(userID)
	defer h.social.Hub().Unsubscribe(userID, ch)

	flusher, ok := c.Writer.(http.Flusher)

	if !ok {

		c.Status(http.StatusInternalServerError)
		return

	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := c.Request.Context().Done()

	for {

		select {

			case <-done:

				return

			case event := <-ch:

				data, _ := json.Marshal(event)
				fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Type, data)

				flusher.Flush()

			case <-ticker.C:

				fmt.Fprintf(c.Writer, ": heartbeat\n\n")

				flusher.Flush()

			}

	}

}
