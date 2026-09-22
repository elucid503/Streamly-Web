package catalog

import (
	"net/http"
	"strconv"

	discover "streamly/internal/features/catalog/discovery"
	"streamly/internal/httpx"

	"github.com/gin-gonic/gin"
)

type FeedHandler struct {

	feed *discover.Service

}

func NewFeedHandler(feed *discover.Service) *FeedHandler {

	return &FeedHandler{

		feed: feed,

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

		httpx.WriteError(c, http.StatusBadRequest, "kind must be movie or show")
		return

	}

	tmdbID, _ := strconv.Atoi(c.Query("tmdbId"))

	if tmdbID <= 0 {

		httpx.WriteError(c, http.StatusBadRequest, "tmdbId required")
		return

	}

	year, _ := strconv.Atoi(c.Query("year"))
	title := c.Query("title")

	result, err := h.feed.Resolve(kind, tmdbID, title, year)

	if err != nil {

		httpx.WriteError(c, http.StatusNotFound, err.Error())
		return

	}

	c.JSON(http.StatusOK, result)

}

func (h *FeedHandler) serve(c *gin.Context, kind string) {

	feed := h.feed.Feed(kind)

	if feed.Sections == nil {

		feed.Sections = []discover.FeedSection{}

	}

	c.JSON(http.StatusOK, feed)

}
