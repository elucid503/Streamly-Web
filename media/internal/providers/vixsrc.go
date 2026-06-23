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
	vixTokenRE    = regexp.MustCompile(`token["']\s*:\s*["']([^"']+)`)
	vixExpiresRE  = regexp.MustCompile(`expires["']\s*:\s*["']([^"']+)`)
	vixPlaylistRE = regexp.MustCompile(`url\s*:\s*["']([^"']+)`)
	vixResRE      = regexp.MustCompile(`RESOLUTION=\d+x(\d+)`)
)

type vixsrcProvider struct{}

func newVixsrc() Provider { return &vixsrcProvider{} }

func (v *vixsrcProvider) Name() string { return "Vixsrc" }

func (v *vixsrcProvider) Fetch(tmdbID int, mediaType string, season, episode int) ([]Stream, error) {

	var apiURL string

	if mediaType == "movie" {

		apiURL = fmt.Sprintf("%s/api/movie/%d", vixsrcBase, tmdbID)

	} else {

		apiURL = fmt.Sprintf("%s/api/tv/%d/%d/%d", vixsrcBase, tmdbID, season, episode)

	}

	vixHeaders := map[string]string{

		"Referer": vixsrcBase,
		"Origin":  vixsrcBase,

	}

	data, err := getJSON(apiURL, vixHeaders)

	if err != nil {

		return nil, fmt.Errorf("vixsrc: api call failed: %w", err)

	}

	src, _ := data["src"].(string)

	if src == "" {

		return nil, fmt.Errorf("vixsrc: no src in response")

	}

	html, err := getText(vixsrcBase+src, map[string]string{

		"Referer": apiURL,
		"Origin":  vixsrcBase,

	})

	if err != nil {

		return nil, fmt.Errorf("vixsrc: embed page failed: %w", err)

	}

	token := matchFirst(vixTokenRE, html)
	expires := matchFirst(vixExpiresRE, html)
	playlist := matchFirst(vixPlaylistRE, html)

	if token == "" || expires == "" || playlist == "" {

		return nil, fmt.Errorf("vixsrc: could not extract token data from embed page")

	}

	exp, _ := strconv.ParseInt(expires, 10, 64)

	if time.Now().Unix() > exp-60 {

		return nil, fmt.Errorf("vixsrc: token expired")

	}

	sep := "?"

	if strings.Contains(playlist, "?") {

		sep = "&"

	}

	masterURL := fmt.Sprintf("%s%stoken=%s&expires=%s&h=1", playlist, sep, token, expires)

	m3u8, err := getText(masterURL, map[string]string{"Referer": apiURL})

	if err != nil {

		return nil, fmt.Errorf("vixsrc: failed to fetch playlist: %w", err)

	}

	matches := vixResRE.FindAllStringSubmatch(m3u8, -1)
	best := 0

	for _, m := range matches {

		if h, _ := strconv.Atoi(m[1]); h > best {

			best = h

		}

	}

	if best == 0 {

		return nil, fmt.Errorf("vixsrc: no valid resolution found in playlist")

	}

	return []Stream{

		{

			Name:     fmt.Sprintf("Vixsrc %dp", best),
			URL:      masterURL,
			Quality:  fmt.Sprintf("%dp", best),
			Provider: "Vixsrc",
			IsHLS:    true,
			Headers: map[string]string{

				"Referer":    apiURL,
				"User-Agent": providerUA,

			},

		},

	}, nil

}
