package handlers

import (
	"bufio"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"streamly/internal/services"
)

type ProxyHandler struct {
	proxy *services.ProxyService
}

func NewProxyHandler(proxy *services.ProxyService) *ProxyHandler {
	return &ProxyHandler{proxy: proxy}
}

func (h *ProxyHandler) Serve(c *gin.Context) {
	token := c.Param("token")
	entry, err := h.proxy.ResolveToken(c.Request.Context(), token)
	if err != nil {
		writeError(c, http.StatusNotFound, "stream session expired or not found")
		return
	}

	if len(entry.InlineContent) > 0 {
		contentType := entry.InlineContentType
		if contentType == "" {
			contentType = "text/plain; charset=utf-8"
		}
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, contentType, entry.InlineContent)
		return
	}

	ctx := c.Request.Context()
	resp, err := h.proxy.Fetch(ctx, entry, c.Request.Header)
	if err != nil {
		resp, err = h.proxy.Fetch(ctx, entry, c.Request.Header)
	}
	if err != nil {
		if services.IsClientDisconnect(err) || ctx.Err() != nil {
			return
		}
		writeError(c, http.StatusBadGateway, "upstream stream unavailable")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		writeError(c, http.StatusBadGateway, "upstream returned error")
		return
	}

	contentType := services.DetectContentType(entry.TargetURL, resp.Header)
	reader := bufio.NewReader(resp.Body)

	if services.IsPlaylist(contentType, entry.TargetURL) || isM3U8Peek(reader) {
		body, err := io.ReadAll(reader)
		if err != nil {
			writeError(c, http.StatusBadGateway, "failed to read playlist")
			return
		}
		rewritten := h.proxy.RewritePlaylist(body, entry, baseURL(c))
		c.Data(http.StatusOK, "application/vnd.apple.mpegurl", rewritten)
		return
	}

	resp.Body = io.NopCloser(reader)
	if err := services.ForwardMediaResponse(c.Writer, resp); err != nil {
		if !services.IsClientDisconnect(err) {
			_ = c.Error(err)
		}
	}
}

func isM3U8Peek(reader *bufio.Reader) bool {
	peek, err := reader.Peek(7)
	if err != nil || len(peek) < 7 {
		return false
	}
	return strings.HasPrefix(string(peek), "#EXTM3U")
}