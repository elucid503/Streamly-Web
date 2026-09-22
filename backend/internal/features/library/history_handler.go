package library

import (
	"net/http"
	"strconv"

	"streamly/internal/httpx"
	"streamly/internal/middleware"
	"streamly/internal/models"

	"github.com/gin-gonic/gin"
)

type HistoryHandler struct {

	history *HistoryService

}

func NewHistoryHandler(history *HistoryService) *HistoryHandler {

	return &HistoryHandler{history: history}

}

func (h *HistoryHandler) List(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	var mediaID *int

	if raw := c.Query("mediaId"); raw != "" {

		if id, err := strconv.Atoi(raw); err == nil {

			mediaID = &id

		}

	}

	items, err := h.history.List(c.Request.Context(), userID, limit, mediaID)

	if err != nil {

		httpx.HandleError(c, err)
		return

	}

	if items == nil {

		items = []models.WatchHistoryItem{}

	}

	c.JSON(http.StatusOK, items)

}

func (h *HistoryHandler) Upsert(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	var input HistoryUpsert

	if err := c.ShouldBindJSON(&input); err != nil {

		httpx.WriteError(c, http.StatusBadRequest, "invalid request")
		return

	}

	item, err := h.history.Upsert(c.Request.Context(), userID, input)

	if err != nil {

		httpx.HandleError(c, err)
		return

	}

	c.JSON(http.StatusOK, item)

}

func (h *HistoryHandler) Delete(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	if err := h.history.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {

		httpx.HandleError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}
