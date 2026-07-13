package handlers

import (
	"net/http"
	"strconv"

	"streamly/internal/middleware"
	"streamly/internal/services"
	"streamly/internal/services/discover"

	"github.com/gin-gonic/gin"
)

type FeedHandler struct {

	feed *discover.Service
	history *services.HistoryService
	favorites *services.FavoritesService

}

func NewFeedHandler(feed *discover.Service, history *services.HistoryService, favorites *services.FavoritesService) *FeedHandler {

	return &FeedHandler{

		feed: feed,
		history: history,
		favorites: favorites,

	}

}

func (h *FeedHandler) Movies(c *gin.Context) {

	h.serve(c, "movie")

}

func (h *FeedHandler) Shows(c *gin.Context) {

	h.serve(c, "show")

}

func (h *FeedHandler) Resolve(c *gin.Context) {

	kind := c.Query("kind")

	if kind != "movie" && kind != "show" {

		writeError(c, http.StatusBadRequest, "kind must be movie or show")
		return

	}

	tmdbID, _ := strconv.Atoi(c.Query("tmdbId"))

	if tmdbID <= 0 {

		writeError(c, http.StatusBadRequest, "tmdbId required")
		return

	}

	year, _ := strconv.Atoi(c.Query("year"))
	title := c.Query("title")

	result, err := h.feed.Resolve(kind, tmdbID, title, year)

	if err != nil {

		writeError(c, http.StatusNotFound, err.Error())
		return

	}

	c.JSON(http.StatusOK, result)

}

func (h *FeedHandler) serve(c *gin.Context, kind string) {

	userID := c.GetString(middleware.UserIDKey)

	history, err := h.history.List(c.Request.Context(), userID, 50, nil)

	if err != nil {

		history = nil

	}

	favorites, err := h.favorites.List(c.Request.Context(), userID)

	if err != nil {

		favorites = nil

	}

	feed := h.feed.Feed(kind, history, favorites)

	if feed.Sections == nil {

		feed.Sections = []discover.FeedSection{}

	}

	c.JSON(http.StatusOK, feed)

}
