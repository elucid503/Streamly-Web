package providers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const vixsrcBase = "https://vixsrc.to"

var (
	vixTokenRe    = regexp.MustCompile(`token["']\s*:\s*["']([^"']+)`)
	vixExpiresRe  = regexp.MustCompile(`expires["']\s*:\s*["']([^"']+)`)
	vixPlaylistRe = regexp.MustCompile(`url\s*:\s*["']([^"']+)`)
	vixResRE      = regexp.MustCompile(`RESOLUTION=\d+x(\d+)`)
)

type vixsrcProvider struct {

	baseURL string

}

func newVixsrc() Provider {

	return &vixsrcProvider{baseURL: vixsrcBase}

}

func (v *vixsrcProvider) Name() string { return "Vixsrc" }

func (v *vixsrcProvider) origin() string {

	return v.baseURL

}

func (v *vixsrcProvider) Fetch(tmdbID int, mediaType string, season, episode int) ([]Stream, error) {

	return v.fetchAtBase(v.baseURL, tmdbID, mediaType, season, episode)

}

func (v *vixsrcProvider) fetchAtBase(base string, tmdbID int, mediaType string, season, episode int) ([]Stream, error) {

	if tmdbID <= 0 {

		return nil, fmt.Errorf("vixsrc: invalid tmdb id")

	}

	apiURL, err := vixsrcAPIURL(base, tmdbID, mediaType, season, episode)

	if err != nil {

		return nil, err

	}

	vixHeaders := map[string]string{

		"Referer": base,
		"Origin":  base,

	}

	streamDebugf("vixsrc api GET %s", apiURL)

	data, err := getJSON(apiURL, vixHeaders)

	if err != nil {

		streamDebugf("vixsrc api failed: %v", err)

		return nil, fmt.Errorf("vixsrc: api call failed: %w", err)

	}

	src, _ := data["src"].(string)

	if src == "" {

		streamDebugf("vixsrc api missing src field in %+v", data)

		return nil, fmt.Errorf("vixsrc: no src in response")

	}

	streamDebugf("vixsrc embed GET %s%s", base, src)

	html, err := getText(base+src, map[string]string{

		"Referer": apiURL,
		"Origin":  base,

	})

	if err != nil {

		streamDebugf("vixsrc embed failed: %v", err)

		return nil, fmt.Errorf("vixsrc: embed page failed: %w", err)

	}

	token := matchFirst(vixTokenRe, html)
	expires := matchFirst(vixExpiresRe, html)
	playlist := matchFirst(vixPlaylistRe, html)

	if token == "" || expires == "" || playlist == "" {

		streamDebugf("vixsrc embed parse failed token=%t expires=%t playlist=%t html_bytes=%d", token != "", expires != "", playlist != "", len(html))

		return nil, fmt.Errorf("vixsrc: could not extract token data from embed page")

	}

	exp, _ := strconv.ParseInt(expires, 10, 64)

	if time.Now().Unix() > exp-60 {

		return nil, fmt.Errorf("vixsrc: token expired")

	}

	masterURL := appendVixsrcToken(playlist, token, expires, true)

	streamDebugf("vixsrc master GET %s", masterURL)

	m3u8, err := getText(masterURL, map[string]string{

		"Referer": apiURL,
		"Origin":  base,

	})

	if err != nil {

		streamDebugf("vixsrc master failed: %v", err)

		return nil, fmt.Errorf("vixsrc: failed to fetch playlist: %w", err)

	}

	variants, err := parseVixsrcMasterPlaylist(m3u8)

	if err != nil {

		return nil, err

	}

	playbackHeaders := map[string]string{

		"Referer": apiURL,
		"Origin":  base,

	}

	// Serve the master playlist for every quality entry. Variant URLs are
	// video-only renditions; the master manifest carries separate audio tracks.
	streams := make([]Stream, 0, len(variants))

	for _, variant := range variants {

		streams = append(streams, Stream{

			Name:     fmt.Sprintf("Vixsrc %dp", variant.Height),
			URL:      masterURL,
			Quality:  fmt.Sprintf("%dp", variant.Height),
			Provider: "Vixsrc",
			IsHLS:    true,
			Headers:  playbackHeaders,

		})

	}

	return streams, nil

}

func vixsrcAPIURL(base string, tmdbID int, mediaType string, season, episode int) (string, error) {

	switch mediaType {

	case "movie":

		return fmt.Sprintf("%s/api/movie/%d", base, tmdbID), nil

	case "tv", "series":

		if season <= 0 || episode <= 0 {

			return "", fmt.Errorf("vixsrc: invalid season/episode")

		}

		return fmt.Sprintf("%s/api/tv/%d/%d/%d", base, tmdbID, season, episode), nil

	default:

		return "", fmt.Errorf("vixsrc: unsupported media type %q", mediaType)

	}

}

func appendVixsrcToken(playlist, token, expires string, includeHeight bool) string {

	sep := "?"

	if strings.Contains(playlist, "?") {

		sep = "&"

	}

	url := fmt.Sprintf("%s%stoken=%s&expires=%s", playlist, sep, token, expires)

	if includeHeight {

		url += "&h=1"

	}

	return url

}