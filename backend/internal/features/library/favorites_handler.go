package library

import (
	"net/http"

	"streamly/internal/httpx"
	"streamly/internal/middleware"
	"streamly/internal/models"

	"github.com/gin-gonic/gin"
)

type FavoritesHandler struct {

	favorites *FavoritesService

}

func NewFavoritesHandler(favorites *FavoritesService) *FavoritesHandler {

	return &FavoritesHandler{favorites: favorites}

}

func (h *FavoritesHandler) List(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	items, err := h.favorites.List(c.Request.Context(), userID)

	if err != nil {

		httpx.HandleError(c, err)
		return

	}

	if items == nil {

		items = []models.FavoriteItem{}

	}

	c.JSON(http.StatusOK, items)

}

func (h *FavoritesHandler) Upsert(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	var input FavoriteUpsert

	if err := c.ShouldBindJSON(&input); err != nil {

		httpx.WriteError(c, http.StatusBadRequest, "invalid request")
		return

	}

	item, err := h.favorites.Upsert(c.Request.Context(), userID, input)

	if err != nil {

		httpx.HandleError(c, err)
		return

	}

	c.JSON(http.StatusOK, item)

}

func (h *FavoritesHandler) Delete(c *gin.Context) {

	userID := c.GetString(middleware.UserIDKey)

	if err := h.favorites.Delete(c.Request.Context(), userID, c.Param("kind"), c.Param("key")); err != nil {

		httpx.HandleError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}
