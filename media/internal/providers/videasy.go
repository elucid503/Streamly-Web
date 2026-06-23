package providers

import (

	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

)

const (
	videasyOrigin  = "https://player.videasy.net"
	videasyDecrypt = "https://enc-dec.app/api/dec-videasy"
)

var videasyServers = []struct {
	name     string
	url      string
	movieOnly bool
}{

	{"Neon", "https://api.videasy.net/myflixerzupcloud/sources-with-title", false},
	{"Cypher", "https://api.videasy.net/moviebox/sources-with-title", false},
	{"Reyna", "https://api.videasy.net/primewire/sources-with-title", false},
	{"Omen", "https://api.videasy.net/onionplay/sources-with-title", false},
	{"Breach", "https://api.videasy.net/m4uhd/sources-with-title", false},
	{"Ghost", "https://api.videasy.net/primesrcme/sources-with-title", false},
	{"Sage", "https://api.videasy.net/1movies/sources-with-title", false},
	{"Vyse", "https://api.videasy.net/hdmovie/sources-with-title", false},
	{"Raze", "https://api.videasy.net/superflix/sources-with-title", false},
	{"Yoru", "https://api.videasy.net/cdn/sources-with-title", true},

}

type videasyProvider struct {
	tmdbKey string
}

func newVideasy(tmdbKey string) Provider {
	return &videasyProvider{tmdbKey: tmdbKey}
}

func (v *videasyProvider) Name() string { return "Videasy" }

type videasyTMDBDetails struct {
	title  string
	year   string
	imdbID string
	kind   string
}

func (v *videasyProvider) tmdbDetails(tmdbID int, mediaType string) (videasyTMDBDetails, error) {

	kind := "movie"

	if mediaType == "tv" || mediaType == "series" {

		kind = "tv"

	}

	apiURL := fmt.Sprintf("https://api.themoviedb.org/3/%s/%d?api_key=%s&append_to_response=external_ids", kind, tmdbID, v.tmdbKey)

	data, err := getJSON(apiURL, nil)

	if err != nil {

		return videasyTMDBDetails{}, err

	}

	title, _ := data["title"].(string)

	if title == "" {

		title, _ = data["name"].(string)

	}

	releaseDate, _ := data["release_date"].(string)

	if releaseDate == "" {

		releaseDate, _ = data["first_air_date"].(string)

	}

	year := ""

	if len(releaseDate) >= 4 {

		year = releaseDate[:4]

	}

	imdbID := ""

	if ext, ok := data["external_ids"].(map[string]any); ok {

		imdbID, _ = ext["imdb_id"].(string)

	}

	return videasyTMDBDetails{title: title, year: year, imdbID: imdbID, kind: kind}, nil

}

func (v *videasyProvider) Fetch(tmdbID int, mediaType string, season, episode int) ([]Stream, error) {

	details, err := v.tmdbDetails(tmdbID, mediaType)

	if err != nil {

		return nil, fmt.Errorf("videasy: TMDB lookup failed: %w", err)

	}

	if details.title == "" {

		return nil, fmt.Errorf("videasy: no title from TMDB for %d", tmdbID)

	}

	headers := map[string]string{

		"Origin":  videasyOrigin,
		"Referer": videasyOrigin + "/",

	}

	type serverResult struct {
		streams []Stream
	}

	ch := make(chan serverResult, len(videasyServers))

	var wg sync.WaitGroup

	for _, srv := range videasyServers {

		if srv.movieOnly && (mediaType == "tv" || mediaType == "series") {

			continue

		}

		wg.Add(1)

		go func(srvName, srvURL string) {

			defer wg.Done()

			params := url.Values{}
			params.Set("title", details.title)
			params.Set("mediaType", details.kind)
			params.Set("year", details.year)
			params.Set("tmdbId", fmt.Sprintf("%d", tmdbID))
			params.Set("imdbId", details.imdbID)

			if mediaType == "tv" || mediaType == "series" {

				params.Set("seasonId", fmt.Sprintf("%d", season))
				params.Set("episodeId", fmt.Sprintf("%d", episode))

			}

			raw, err := getJSONRaw(srvURL+"?"+params.Encode(), headers)

			if err != nil {

				ch <- serverResult{}
				return

			}

			encText := strings.TrimSpace(string(raw))

			if encText == "" || len(encText) < 20 || strings.HasPrefix(encText, "<") {

				ch <- serverResult{}
				return

			}

			decBody := map[string]any{

				"text": encText,
				"id":   fmt.Sprintf("%d", tmdbID),

			}

			decRaw, err := postJSON(videasyDecrypt, decBody, nil)

			if err != nil {

				ch <- serverResult{}
				return

			}

			var decResp map[string]any

			if err := json.Unmarshal(decRaw, &decResp); err != nil {

				ch <- serverResult{}
				return

			}

			resData, _ := decResp["result"].(map[string]any)

			if resData == nil {

				resData = decResp

			}

			sources, _ := resData["sources"].([]any)

			var streams []Stream

			for _, s := range sources {

				src, _ := s.(map[string]any)

				if src == nil {

					continue

				}

				u, _ := src["url"].(string)

				if u == "" {

					continue

				}

				qual, _ := src["quality"].(string)

				if qual == "" {

					qual = "Auto"

				}

				streams = append(streams, Stream{

					Name:     fmt.Sprintf("Videasy %s", srvName),
					URL:      u,
					Quality:  qual,
					Provider: "Videasy",
					Headers: map[string]string{

						"Referer": videasyOrigin + "/",
						"Origin":  videasyOrigin,

					},

				})

			}

			ch <- serverResult{streams: streams}

		}(srv.name, srv.url)

	}

	go func() {

		wg.Wait()
		close(ch)

	}()

	seen := make(map[string]struct{})
	var all []Stream

	for res := range ch {

		for _, s := range res.streams {

			if _, ok := seen[s.URL]; ok {

				continue

			}

			seen[s.URL] = struct{}{}
			all = append(all, s)

		}

	}

	return all, nil

}
