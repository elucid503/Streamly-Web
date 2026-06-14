package handlers

import (
	"net/http"

	"streamly/internal/middleware"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {

	settings *services.SettingsService

}

func NewSettingsHandler(settings *services.SettingsService) *SettingsHandler {

	return &SettingsHandler{settings: settings}

}

func (h *SettingsHandler) Get(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	settings, err := h.settings.Get(c.Request.Context(), userID)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, settings)

}

func (h *SettingsHandler) Update(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	var update services.SettingsUpdate

	if err := c.ShouldBindJSON(&update); err != nil {

		writeError(c, http.StatusBadRequest, "invalid request")
		return

	}

	settings, err := h.settings.Update(c.Request.Context(), userID, update)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, settings)

}
