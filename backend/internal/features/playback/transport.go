package playback

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
)

func IsClientDisconnect(err error) bool {

	if err == nil {

		return false

	}

	if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {

		return true

	}

	if errors.Is(err, net.ErrClosed) {

		return true

	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "use of closed network connection")

}

func ForwardMediaResponse(dst http.ResponseWriter, resp *http.Response, targetURL string) error {

	for key, values := range resp.Header {

		if strings.EqualFold(key, "Transfer-Encoding") || strings.EqualFold(key, "Content-Type") {

			continue

		}

		for _, value := range values {

			dst.Header().Add(key, value)

		}

	}

	dst.Header().Set("Content-Type", DetectContentType(targetURL, resp.Header))

	// Claiming seekability an origin does not have makes AVPlayer request a
	// range, get the whole file back, and give up on the media.
	rangeIgnored := resp.Request != nil && resp.Request.Header.Get("Range") != "" && resp.StatusCode == http.StatusOK

	if rangeIgnored {

		dst.Header().Del("Accept-Ranges")

	} else if dst.Header().Get("Accept-Ranges") == "" {

		dst.Header().Set("Accept-Ranges", "bytes")

	}

	dst.Header().Set("Cache-Control", "no-store")

	dst.WriteHeader(resp.StatusCode)

	_, err := io.Copy(dst, resp.Body)

	if IsClientDisconnect(err) {

		return nil

	}

	return err

}

// DetectContentType prefers a specific upstream type but falls back to the URL
// extension. Origins routinely label segments application/octet-stream or
// text/html; hls.js ignores that, AVFoundation on an AirPlay receiver does not.
func DetectContentType(urlStr string, header http.Header) string {

	ct := strings.TrimSpace(header.Get("Content-Type"))

	if ct != "" && !isGenericContentType(ct) {

		return ct

	}

	byExt := contentTypeByExtension(urlStr)

	if byExt != "" {

		return byExt

	}

	if ct != "" {

		return ct

	}

	return "application/octet-stream"

}

func isGenericContentType(ct string) bool {

	switch strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0])) {

	case "application/octet-stream", "binary/octet-stream", "application/binary", "text/plain", "text/html", "application/force-download":

		return true

	}

	return false

}

func contentTypeByExtension(urlStr string) string {

	path := strings.ToLower(strings.Split(strings.Split(urlStr, "#")[0], "?")[0])

	dot := strings.LastIndex(path, ".")

	if dot < 0 {

		return ""

	}

	switch path[dot:] {

	case ".m3u8", ".m3u":

		return "application/vnd.apple.mpegurl"

	case ".ts", ".mts":

		return "video/mp2t"

	case ".mp4", ".m4s", ".m4v", ".cmfv", ".fmp4":

		return "video/mp4"

	case ".m4a", ".cmfa":

		return "audio/mp4"

	case ".aac":

		return "audio/aac"

	case ".mp3":

		return "audio/mpeg"

	case ".webm":

		return "video/webm"

	case ".vtt":

		return "text/vtt"

	case ".key":

		return "application/octet-stream"

	}

	return ""

}

func IsPlaylist(contentType, urlStr string) bool {

	ct := strings.ToLower(contentType)

	if strings.Contains(ct, "mpegurl") || strings.Contains(ct, "m3u") {

		return true

	}

	lower := strings.ToLower(strings.Split(urlStr, "?")[0])

	return strings.HasSuffix(lower, ".m3u8") || strings.HasSuffix(lower, ".m3u")

}
