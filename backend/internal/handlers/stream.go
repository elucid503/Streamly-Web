package handlers

import (
	"net/http"
	"strconv"

	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type StreamHandler struct {

	media *services.MediaService
	proxy *services.ProxyService

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

	qualities, err := h.media.MovieQualities(id)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	stream := services.BuildStreamDTO(qualities)

	if stream == nil {

		writeError(c, http.StatusNotFound, "no stream available")
		return

	}

	stream.Qualities = h.proxyHeaderQualities(c, stream.Qualities)

	c.JSON(http.StatusOK, stream)

}

func (h *StreamHandler) MovieSubtitles(c *gin.Context) {

	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {

		writeError(c, http.StatusBadRequest, "invalid id")
		return

	}

	tracks := h.subtitles.MovieTracks(c.Request.Context(), id)

	if tracks == nil {

		tracks = []services.SubtitleDTO{}

	}

	c.JSON(http.StatusOK, tracks)

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

	qualities, err := h.media.EpisodeQualities(showID, season, episode)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	stream := services.BuildStreamDTO(qualities)

	if stream == nil {

		writeError(c, http.StatusNotFound, "no stream available")
		return

	}

	stream.Qualities = h.proxyHeaderQualities(c, stream.Qualities)

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

	tracks := h.subtitles.EpisodeTracks(c.Request.Context(), showID, season, episode)

	if tracks == nil {

		tracks = []services.SubtitleDTO{}

	}

	c.JSON(http.StatusOK, tracks)

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

	channel := stream.Channel

	if cached, ok := h.media.LiveChannel(daddyID); ok {

		c.JSON(http.StatusOK, gin.H{

			"proxyUrl": baseURL(c) + session.ProxyPath,
			"isHls": true,
			"channel": cached,

		})

		return

	}

	c.JSON(http.StatusOK, gin.H{

		"proxyUrl": baseURL(c) + session.ProxyPath,
		"isHls": true,
		"channel": channel,

	})

}

// proxyHeaderQualities replaces gated qualities with same-origin proxy URLs.
// Direct Febbox progressive URLs without headers are returned unchanged.
func (h *StreamHandler) proxyHeaderQualities(c *gin.Context, qualities []services.QualityDTO) []services.QualityDTO {

	base := baseURL(c)

	out := make([]services.QualityDTO, 0, len(qualities))

	for _, q := range qualities {

		if len(q.Headers) > 0 {

			if session, err := h.proxy.CreateSessionWithHeaders(c.Request.Context(), q.URL, q.Headers, q.IsHLS); err == nil {

				proxyURL := base + session.ProxyPath

				q.ProxyURL = proxyURL
				q.URL = proxyURL
				q.Headers = nil

			}

		}

		out = append(out, q)

	}

	return out

}
