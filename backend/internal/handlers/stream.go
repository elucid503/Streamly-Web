package handlers

import (
	"net/http"
	"strconv"

	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type StreamHandler struct {
	media     *services.MediaService
	proxy     *services.ProxyService
	subtitles *services.SubtitleResolver
}

func NewStreamHandler(media *services.MediaService, proxy *services.ProxyService, subtitles *services.SubtitleResolver) *StreamHandler {
	return &StreamHandler{media: media, proxy: proxy, subtitles: subtitles}
}

func (h *StreamHandler) MovieStream(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid id")
		return
	}

	height, _ := strconv.Atoi(c.DefaultQuery("height", strconv.Itoa(h.media.DefaultHeight())))
	qualities, best, err := h.media.MovieQualities(id, height)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if best == nil || best.URL == "" {
		writeError(c, http.StatusNotFound, "no stream available")
		return
	}

	stream := services.BuildStreamDTO(qualities, best)
	if stream == nil {
		writeError(c, http.StatusNotFound, "no stream available")
		return
	}
	if err := h.proxy.AttachProxyURLs(c.Request.Context(), stream, "", baseURL(c)); err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, stream)
}

func (h *StreamHandler) MovieSubtitles(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid id")
		return
	}
	c.JSON(http.StatusOK, h.subtitles.MovieTracks(c.Request.Context(), baseURL(c), id))
}

func (h *StreamHandler) EpisodeStream(c *gin.Context) {
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

	height, _ := strconv.Atoi(c.DefaultQuery("height", strconv.Itoa(h.media.DefaultHeight())))
	qualities, best, err := h.media.EpisodeQualities(showID, season, episode, height)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if best == nil || best.URL == "" {
		writeError(c, http.StatusNotFound, "no stream available")
		return
	}

	stream := services.BuildStreamDTO(qualities, best)
	if stream == nil {
		writeError(c, http.StatusNotFound, "no stream available")
		return
	}
	if err := h.proxy.AttachProxyURLs(c.Request.Context(), stream, "", baseURL(c)); err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, stream)
}

func (h *StreamHandler) EpisodeSubtitles(c *gin.Context) {
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
	c.JSON(http.StatusOK, h.subtitles.EpisodeTracks(c.Request.Context(), baseURL(c), showID, season, episode))
}

func (h *StreamHandler) MovieIntro(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid id")
		return
	}
	durationMs, _ := strconv.ParseInt(c.DefaultQuery("durationMs", "0"), 10, 64)
	intro, err := h.media.MovieIntro(id, durationMs)
	if err != nil {
		c.JSON(http.StatusOK, &services.IntroDTO{})
		return
	}
	c.JSON(http.StatusOK, intro)
}

func (h *StreamHandler) EpisodeIntro(c *gin.Context) {
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
	durationMs, _ := strconv.ParseInt(c.DefaultQuery("durationMs", "0"), 10, 64)
	intro, err := h.media.EpisodeIntro(showID, season, episode, durationMs)
	if err != nil {
		c.JSON(http.StatusOK, &services.IntroDTO{})
		return
	}
	c.JSON(http.StatusOK, intro)
}

func (h *StreamHandler) NextEpisode(c *gin.Context) {
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

	next, err := h.media.NextEpisode(showID, season, episode)
	if err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	if next == nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, next)
}

func (h *StreamHandler) LiveStream(c *gin.Context) {
	daddyID := c.Param("id")
	stream, err := h.media.ResolveLiveStream(daddyID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	session, err := h.proxy.CreateSession(c.Request.Context(), stream.URL, stream.Referer, true)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"proxyUrl": baseURL(c) + session.ProxyPath,
		"isHls":    true,
		"channel":  stream.Channel,
	})
}
