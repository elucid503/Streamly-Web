package handlers

import (
	"net/http"
	"strconv"

	mediakit "mediakit"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type CatalogHandler struct {

	media *services.MediaService

}

func NewCatalogHandler(media *services.MediaService) *CatalogHandler {

	return &CatalogHandler{media: media}

}

func (h *CatalogHandler) Search(c *gin.Context) {

	query := c.Query("q")

	if query == "" {

		writeError(c, http.StatusBadRequest, "query required")
		return

	}

	results, err := h.media.Search(query)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(results))

}

func (h *CatalogHandler) MovieTrending(c *gin.Context) {

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	items, err := h.media.TrendingHits(mediakit.MediaMovie, limit)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(items))

}

func (h *CatalogHandler) ShowTrending(c *gin.Context) {

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	items, err := h.media.TrendingHits(mediakit.MediaShow, limit)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(items))

}

func (h *CatalogHandler) MovieCategories(c *gin.Context) {

	cats, err := h.media.Categories(mediakit.MediaMovie)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(cats))

}

func (h *CatalogHandler) ShowCategories(c *gin.Context) {

	cats, err := h.media.Categories(mediakit.MediaShow)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(cats))

}

func (h *CatalogHandler) MovieCategoryTitles(c *gin.Context) {

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))

	titles, err := h.media.CategoryTitles(mediakit.MediaMovie, c.Param("id"), page, limit)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(titles))

}

func (h *CatalogHandler) ShowCategoryTitles(c *gin.Context) {

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))

	titles, err := h.media.CategoryTitles(mediakit.MediaShow, c.Param("id"), page, limit)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(titles))

}

func (h *CatalogHandler) MovieDetails(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid id")
		return

	}

	details, err := h.media.MovieDetails(id)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, details)

}

func (h *CatalogHandler) ShowDetails(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid id")
		return

	}

	details, err := h.media.ShowDetails(id)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, details)

}

func (h *CatalogHandler) ShowSeasons(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid id")
		return

	}

	seasons, err := h.media.ShowSeasons(id)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(seasons))

}

func (h *CatalogHandler) SeasonEpisodes(c *gin.Context) {

	showID, err := strconv.Atoi(c.Param("id"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid show id")
		return

	}

	season, err := strconv.Atoi(c.Param("season"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid season")
		return

	}

	episodes, err := h.media.SeasonEpisodes(showID, season)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(episodes))

}

func (h *CatalogHandler) EpisodeDetails(c *gin.Context) {

	showID, err := strconv.Atoi(c.Param("id"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid show id")
		return

	}

	season, err := strconv.Atoi(c.Param("season"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid season")
		return

	}

	episode, err := strconv.Atoi(c.Param("episode"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid episode")
		return

	}

	details, err := h.media.EpisodeDetails(showID, season, episode)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, details)

}

func (h *CatalogHandler) LiveChannels(c *gin.Context) {

	channels, err := h.media.LiveChannels()

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(channels))

}

func (h *CatalogHandler) LivePopular(c *gin.Context) {

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))

	channels, err := h.media.LivePopular(limit)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(channels))

}

func (h *CatalogHandler) LiveSearch(c *gin.Context) {

	query := c.Query("q")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "48"))

	channels, err := h.media.LiveSearch(query, limit)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusOK, jsonSlice(channels))

}
