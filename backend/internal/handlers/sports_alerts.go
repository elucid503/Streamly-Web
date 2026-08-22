package handlers

import (
	"errors"
	"net/http"

	"streamly/internal/middleware"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type SportsAlertsHandler struct {

	alerts *services.SportsAlertsService
	push *services.PushService

}

func NewSportsAlertsHandler(alerts *services.SportsAlertsService, push *services.PushService) *SportsAlertsHandler {

	return &SportsAlertsHandler{alerts: alerts, push: push}

}

func (h *SportsAlertsHandler) VapidPublicKey(c *gin.Context) {

	if !h.push.Configured() {

		writeError(c, http.StatusServiceUnavailable, "push notifications are not configured")
		return

	}

	c.JSON(http.StatusOK, gin.H{"publicKey": h.push.PublicKey()})

}

func (h *SportsAlertsHandler) UpsertSubscription(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	var input services.PushSubscriptionInput

	if err := c.ShouldBindJSON(&input); err != nil {

		writeError(c, http.StatusBadRequest, "invalid request")
		return

	}

	if err := h.push.UpsertSubscription(c.Request.Context(), userID, input); err != nil {

		handleSportsAlertError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}

type deleteSubscriptionRequest struct {

	Endpoint string `json:"endpoint"`

}

func (h *SportsAlertsHandler) DeleteSubscription(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	var input deleteSubscriptionRequest

	if err := c.ShouldBindJSON(&input); err != nil {

		writeError(c, http.StatusBadRequest, "invalid request")
		return

	}

	if err := h.push.DeleteSubscription(c.Request.Context(), userID, input.Endpoint); err != nil {

		handleSportsAlertError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}

func (h *SportsAlertsHandler) List(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	items, err := h.alerts.List(c.Request.Context(), userID)

	if err != nil {

		handleSportsAlertError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(items))

}

func (h *SportsAlertsHandler) Subscribe(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	item, err := h.alerts.Subscribe(c.Request.Context(), userID, c.Param("matchId"))

	if err != nil {

		handleSportsAlertError(c, err)
		return

	}

	c.JSON(http.StatusOK, item)

}

func (h *SportsAlertsHandler) Unsubscribe(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	if err := h.alerts.Unsubscribe(c.Request.Context(), userID, c.Param("matchId")); err != nil {

		handleSportsAlertError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}

func handleSportsAlertError(c *gin.Context, err error) {

	switch {

	case errors.Is(err, services.ErrPushNotConfigured):

		writeError(c, http.StatusServiceUnavailable, "push notifications are not configured")

	case errors.Is(err, services.ErrInvalidPushSubscription):

		writeError(c, http.StatusBadRequest, "invalid push subscription")

	case errors.Is(err, services.ErrSportsAlertMatch):

		writeError(c, http.StatusNotFound, "match not found")

	default:

		handleServiceError(c, err)

	}

}
