package playback

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"streamly/internal/httpx"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {

	proxy *ProxyService

}

func NewProxyHandler(proxy *ProxyService) *ProxyHandler {

	return &ProxyHandler{proxy: proxy}

}

func (h *ProxyHandler) Serve(c *gin.Context) {

	token := c.Param("token")

	// An AirPlay receiver fetches these URLs itself, with none of the browser's
	// context; STREAM_DEBUG=1 is the only way to see what it actually asked for.
	proxyDebugf("%s %s ua=%q range=%q", c.Request.Method, token, c.GetHeader("User-Agent"), c.GetHeader("Range"))

	entry, err := h.proxy.ResolveToken(token)

	if err != nil {

		proxyDebugf("token %s not found", token)

		httpx.WriteError(c, http.StatusNotFound, "stream session expired or not found")
		return

	}

	ctx := c.Request.Context()

	method := c.Request.Method

	resp, err := h.proxy.Fetch(ctx, entry, method, c.Request.Header)

	if err != nil {

		resp, err = h.proxy.Fetch(ctx, entry, method, c.Request.Header)

	}

	// Origins that reject HEAD would otherwise look like dead media to an
	// AirPlay receiver, which probes with HEAD before it will play anything.
	if method == http.MethodHead && (err != nil || resp.StatusCode >= 400) {

		if resp != nil {

			resp.Body.Close()

		}

		resp, err = h.proxy.Fetch(ctx, entry, http.MethodGet, c.Request.Header)

	}

	if err != nil {

		if IsClientDisconnect(err) || ctx.Err() != nil {

			return

		}

		proxyDebugf("fetch failed target=%s: %v", entry.TargetURL, err)

		httpx.WriteError(c, http.StatusBadGateway, "upstream stream unavailable")
		return

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {

		proxyDebugf("upstream %d target=%s", resp.StatusCode, entry.TargetURL)

		httpx.WriteError(c, http.StatusBadGateway, "upstream returned error")
		return

	}

	contentType := DetectContentType(entry.TargetURL, resp.Header)

	proxyDebugf("%s %s -> %d type=%q (upstream %q) target=%s", method, token, resp.StatusCode, contentType, resp.Header.Get("Content-Type"), entry.TargetURL)

	if method == http.MethodHead {

		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "no-store")

		if IsPlaylist(contentType, entry.TargetURL) {

			// The rewritten playlist is a different length than upstream's.
			c.Header("Content-Length", "")

		} else if length := resp.Header.Get("Content-Length"); length != "" {

			c.Header("Content-Length", length)
			c.Header("Accept-Ranges", "bytes")

		}

		c.Status(http.StatusOK)
		return

	}

	reader := bufio.NewReader(resp.Body)

	if IsPlaylist(contentType, entry.TargetURL) || isM3U8Peek(reader) {

		body, err := io.ReadAll(reader)

		if err != nil {

			httpx.WriteError(c, http.StatusBadGateway, "failed to read playlist")
			return

		}

		rewritten := h.proxy.RewritePlaylist(body, entry, httpx.BaseURL(c))
		c.Data(http.StatusOK, "application/vnd.apple.mpegurl", rewritten)
		return

	}

	resp.Body = io.NopCloser(reader)

	if err := ForwardMediaResponse(c.Writer, resp, entry.TargetURL); err != nil {

		if !IsClientDisconnect(err) {

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

func proxyDebugf(format string, args ...any) {

	switch strings.ToLower(strings.TrimSpace(os.Getenv("STREAM_DEBUG"))) {

	case "1", "true", "yes", "on":

		log.Printf("proxy: "+format, args...)

	}

}
