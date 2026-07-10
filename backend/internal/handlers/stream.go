package handlers

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"streamly/internal/middleware"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type StreamHandler struct {

	media *services.MediaService
	proxy *services.ProxyService
	settings *services.SettingsService

	subtitles *services.SubtitleResolver

}

func NewStreamHandler(media *services.MediaService, proxy *services.ProxyService, settings *services.SettingsService, subtitles *services.SubtitleResolver) *StreamHandler {

	return &StreamHandler{media: media, proxy: proxy, settings: settings, subtitles: subtitles}

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

		streamDebugf("movie %d stream 404: raw_qualities=%d after_dto_filter=0", id, len(qualities))
		writeError(c, http.StatusNotFound, "no stream available")
		return

	}

	stream.Qualities = h.proxyHeaderQualities(c, stream.Qualities)

	if len(stream.Qualities) == 0 {

		streamDebugf("movie %d stream 404: proxy step emptied qualities", id)
		writeError(c, http.StatusNotFound, "no stream available")
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

		streamDebugf("show %d S%02dE%02d stream 404: raw_qualities=%d after_dto_filter=0", showID, season, episode, len(qualities))
		writeError(c, http.StatusNotFound, "no stream available")
		return

	}

	stream.Qualities = h.proxyHeaderQualities(c, stream.Qualities)

	if len(stream.Qualities) == 0 {

		streamDebugf("show %d S%02dE%02d stream 404: proxy step emptied qualities", showID, season, episode)
		writeError(c, http.StatusNotFound, "no stream available")
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

// LiveStream resolves a channel to its HLS playlist URL. By default the
// frontend plays cdnlive URLs directly (CORS allows it). When the user
// enables proxyLiveStreams (e.g. ISP blocks the CDN), playlists and segments
// are routed through /api/proxy instead.
func (h *StreamHandler) LiveStream(c *gin.Context) {

	id := c.Param("id")

	streamURL, err := h.media.ResolveLiveStream(id)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	if h.shouldProxyLiveStreams(c) {

		session, err := h.proxy.CreateSession(c.Request.Context(), streamURL, "", true)

		if err != nil {

			streamDebugf("live %s proxy session failed: %v", id, err)
			writeError(c, http.StatusBadGateway, "failed to create proxy session")
			return

		}

		streamURL = baseURL(c) + session.ProxyPath

	}

	channel, _ := h.media.LiveChannel(id)

	c.JSON(http.StatusOK, gin.H{

		"streamUrl": streamURL,
		"isHls": true,
		"channel": channel,

	})

}

func (h *StreamHandler) shouldProxyLiveStreams(c *gin.Context) bool {

	if h.settings == nil {

		return false

	}

	userID := c.GetString(middleware.UserIDKey)

	if userID == "" {

		return false

	}

	settings, err := h.settings.Get(c.Request.Context(), userID)

	if err != nil || settings == nil {

		return false

	}

	return settings.ProxyLiveStreams

}

// proxyHeaderQualities replaces gated qualities with same-origin proxy URLs.
// Direct Febbox progressive URLs without headers are returned unchanged.
func (h *StreamHandler) proxyHeaderQualities(c *gin.Context, qualities []services.QualityDTO) []services.QualityDTO {

	base := baseURL(c)

	out := make([]services.QualityDTO, 0, len(qualities))

	for _, q := range qualities {

		if len(q.Headers) > 0 {

			session, err := h.proxy.CreateSessionWithHeaders(c.Request.Context(), q.URL, q.Headers, q.IsHLS)

			if err != nil {

				streamDebugf("proxy session failed url=%s: %v", q.URL, err)

			} else {

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

func streamDebugf(format string, args ...any) {

	switch strings.ToLower(strings.TrimSpace(os.Getenv("STREAM_DEBUG"))) {

	case "1", "true", "yes", "on":

		log.Printf("[stream-debug] "+format, args...)

	default:

	}

}
