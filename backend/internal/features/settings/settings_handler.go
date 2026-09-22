package settings

import (
	"net/http"

	"streamly/internal/httpx"
	"streamly/internal/middleware"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {

	settings *SettingsService

}

func NewSettingsHandler(settings *SettingsService) *SettingsHandler {

	return &SettingsHandler{settings: settings}

}

func (h *SettingsHandler) Get(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	settings, err := h.settings.Get(c.Request.Context(), userID)

	if err != nil {

		httpx.HandleError(c, err)
		return

	}

	c.JSON(http.StatusOK, settings)

}

func (h *SettingsHandler) Update(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	var update SettingsUpdate

	if err := c.ShouldBindJSON(&update); err != nil {

		httpx.WriteError(c, http.StatusBadRequest, "invalid request")
		return

	}

	settings, err := h.settings.Update(c.Request.Context(), userID, update)

	if err != nil {

		httpx.HandleError(c, err)
		return

	}

	c.JSON(http.StatusOK, settings)

}
