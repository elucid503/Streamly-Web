package providers

import (

	"encoding/json"
	"fmt"
	"regexp"
	"strings"

)

const vidsrcBase = "https://vidsrc.to"

var (
	vidsrcDataIDRE    = regexp.MustCompile(`(?i)id\s*=\s*["']player-js["'][^>]*data-id\s*=\s*["']([^"']+)["']|data-id\s*=\s*["']([^"']+)["'][^>]*id\s*=\s*["']player-js["']`)
	vidsrcHLSURLRE    = regexp.MustCompile(`(?i)["']?(?:file|src|hls_url|playlist|url)["']?\s*[=:]\s*["']([^"']*\.m3u8[^"']*)`)
	vidsrcMP4URLRE    = regexp.MustCompile(`(?i)["']?(?:file|src|url)["']?\s*[=:]\s*["']([^"']*\.mp4[^"']*)`)
	vidsrcPlaybackRE  = regexp.MustCompile(`"playbackUrl"\s*:\s*"([^"]+)"`)
)

type vidsrcProvider struct{}

func newVidsrc() Provider { return &vidsrcProvider{} }

func (v *vidsrcProvider) Name() string { return "Vidsrc" }

func (v *vidsrcProvider) Fetch(tmdbID int, mediaType string, season, episode int) ([]Stream, error) {

	kind := "movie"

	if mediaType == "tv" || mediaType == "series" {

		kind = "tv"

	}

	var embedURL string

	if kind == "tv" {

		embedURL = fmt.Sprintf("%s/embed/tv/%d/%d/%d", vidsrcBase, tmdbID, season, episode)

	} else {

		embedURL = fmt.Sprintf("%s/embed/movie/%d", vidsrcBase, tmdbID)

	}

	headers := map[string]string{

		"Referer": vidsrcBase + "/",

	}

	html, err := getText(embedURL, headers)

	if err != nil {

		return nil, fmt.Errorf("vidsrc: failed to fetch embed page: %w", err)

	}

	sourceID := extractVidsrcSourceID(html)

	if sourceID == "" {

		return nil, fmt.Errorf("vidsrc: could not extract source ID from embed page")

	}

	serversURL := fmt.Sprintf("%s/ajax/embed/episode/%s/servers", vidsrcBase, sourceID)

	serversRaw, err := getJSONRaw(serversURL, map[string]string{

		"X-Requested-With": "XMLHttpRequest",
		"Referer":          embedURL,

	})

	if err != nil {

		return nil, fmt.Errorf("vidsrc: failed to fetch servers: %w", err)

	}

	servers := parseVidsrcServers(serversRaw)

	if len(servers) == 0 {

		return nil, fmt.Errorf("vidsrc: no servers found")

	}

	type serverResult struct {
		label string
		url   string
	}

	ch := make(chan serverResult, len(servers))

	for _, srv := range servers {

		go func(srv map[string]any) {

			srvURL, _ := srv["url"].(string)

			if srvURL == "" {

				srvURL, _ = srv["link"].(string)

			}

			if srvURL == "" {

				ch <- serverResult{}
				return

			}

			label, _ := srv["title"].(string)

			if label == "" {

				label, _ = srv["name"].(string)

			}

			if label == "" {

				label = "Server"

			}

			streamURL := extractStreamURL(srvURL)

			ch <- serverResult{label: label, url: streamURL}

		}(srv)

	}

	var streams []Stream

	for range servers {

		res := <-ch

		if res.url == "" {

			continue

		}

		streams = append(streams, Stream{

			Name:     fmt.Sprintf("Vidsrc [%s]", res.label),
			URL:      res.url,
			Quality:  "Auto",
			Provider: "Vidsrc",
			Headers:  map[string]string{"Referer": vidsrcBase + "/"},

		})

	}

	return streams, nil

}

func extractVidsrcSourceID(html string) string {

	// Try both attribute orderings in the player-js element.
	if m := vidsrcDataIDRE.FindStringSubmatch(html); len(m) > 0 {

		for _, v := range m[1:] {

			if v != "" {

				return v

			}

		}

	}

	// Fallback: any element with data-id adjacent to player-js text.
	re := regexp.MustCompile(`data-id\s*=\s*["']([^"']+)["']`)

	if m := re.FindStringSubmatch(html); len(m) > 1 {

		return m[1]

	}

	return ""

}

func parseVidsrcServers(raw []byte) []map[string]any {

	var resp map[string]any

	if err := json.Unmarshal(raw, &resp); err != nil {

		return nil

	}

	result := resp["result"]

	if result == nil {

		return nil

	}

	switch v := result.(type) {

	case []any:

		out := make([]map[string]any, 0, len(v))

		for _, item := range v {

			if m, ok := item.(map[string]any); ok {

				out = append(out, m)

			}

		}

		return out

	case map[string]any:

		out := make([]map[string]any, 0, len(v))

		for _, item := range v {

			if m, ok := item.(map[string]any); ok {

				out = append(out, m)

			}

		}

		return out

	}

	return nil

}

func extractStreamURL(pageURL string) string {

	html, err := getText(pageURL, nil)

	if err != nil {

		return ""

	}

	for _, re := range []*regexp.Regexp{vidsrcPlaybackRE, vidsrcHLSURLRE, vidsrcMP4URLRE} {

		if m := re.FindStringSubmatch(html); len(m) > 1 {

			u := strings.TrimSpace(m[1])

			if strings.HasPrefix(u, "http") {

				return u

			}

		}

	}

	// Fallback: scan script blocks for JSON with url/file/hls key.
	scriptRE := regexp.MustCompile(`<script[^>]*>([\s\S]*?)</script>`)

	for _, sm := range scriptRE.FindAllStringSubmatch(html, -1) {

		inner := sm[1]

		urlRE := regexp.MustCompile(`"(?:url|file|src|hls|playlist)"\s*:\s*"(https?://[^"]+\.(?:m3u8|mp4)[^"]*)"`)

		if m := urlRE.FindStringSubmatch(inner); len(m) > 1 {

			return m[1]

		}

	}

	return ""

}
