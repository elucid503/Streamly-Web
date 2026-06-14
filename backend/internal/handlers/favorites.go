package handlers

import (
	"net/http"

	"streamly/internal/middleware"
	"streamly/internal/models"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type FavoritesHandler struct {
	favorites *services.FavoritesService
}

func NewFavoritesHandler(favorites *services.FavoritesService) *FavoritesHandler {

	return &FavoritesHandler{favorites: favorites}

}

func (h *FavoritesHandler) List(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	items, err := h.favorites.List(c.Request.Context(), userID)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	if items == nil {

		items = []models.FavoriteItem{}

	}

	c.JSON(http.StatusOK, items)

}

func (h *FavoritesHandler) Upsert(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	var input services.FavoriteUpsert

	if err := c.ShouldBindJSON(&input); err != nil {

		writeError(c, http.StatusBadRequest, "invalid request")
		return

	}

	item, err := h.favorites.Upsert(c.Request.Context(), userID, input)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, item)

}

func (h *FavoritesHandler) Delete(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	if err := h.favorites.Delete(c.Request.Context(), userID, c.Param("kind"), c.Param("key")); err != nil {

		handleServiceError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}
