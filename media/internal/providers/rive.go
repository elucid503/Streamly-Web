package providers

import (

	"encoding/json"
	"fmt"

)

const riveBase = "https://rivestream.live"

type riveProvider struct {
	tmdbKey string
}

func newRive(tmdbKey string) Provider {

	return &riveProvider{tmdbKey: tmdbKey}

}

func (r *riveProvider) Name() string { return "Rive" }

func (r *riveProvider) resolveIMDB(tmdbID int, kind string) (string, error) {

	apiURL := fmt.Sprintf("https://api.themoviedb.org/3/%s/%d?api_key=%s&append_to_response=external_ids", kind, tmdbID, r.tmdbKey)

	data, err := getJSON(apiURL, nil)

	if err != nil {

		return "", err

	}

	ext, _ := data["external_ids"].(map[string]any)

	if ext == nil {

		return "", fmt.Errorf("rive: no external_ids for tmdb:%d", tmdbID)

	}

	imdbID, _ := ext["imdb_id"].(string)

	if imdbID == "" {

		return "", fmt.Errorf("rive: no imdb_id for tmdb:%d", tmdbID)

	}

	return imdbID, nil

}

func (r *riveProvider) Fetch(tmdbID int, mediaType string, season, episode int) ([]Stream, error) {

	kind := "movie"

	if mediaType == "tv" || mediaType == "series" {

		kind = "tv"

	}

	imdbID, err := r.resolveIMDB(tmdbID, kind)

	if err != nil {

		return nil, fmt.Errorf("rive: TMDB lookup failed: %w", err)

	}

	requestID := "movieLinks"

	if kind == "tv" {

		requestID = "tvLinks"

	}

	var apiURL string

	if kind == "tv" {

		apiURL = fmt.Sprintf("%s/api/backendfetch?requestID=%s&id=%s&season=%d&episode=%d&type=tv",
			riveBase, requestID, imdbID, season, episode)

	} else {

		apiURL = fmt.Sprintf("%s/api/backendfetch?requestID=%s&id=%s&season=0&episode=0&type=movie",
			riveBase, requestID, imdbID)

	}

	headers := map[string]string{

		"Referer": riveBase + "/",
		"Origin":  riveBase,

	}

	raw, err := getJSONRaw(apiURL, headers)

	if err != nil {

		return nil, fmt.Errorf("rive: api call failed: %w", err)

	}

	// Response may be a single object or array of objects.
	var asArray []json.RawMessage

	if json.Unmarshal(raw, &asArray) != nil {

		asArray = []json.RawMessage{raw}

	}

	var streams []Stream

	for i, item := range asArray {

		var obj map[string]any

		if err := json.Unmarshal(item, &obj); err != nil {

			continue

		}

		label, _ := obj["server"].(string)

		if label == "" {

			label, _ = obj["provider"].(string)

		}

		if label == "" {

			label = fmt.Sprintf("S%d", i+1)

		}

		sources := extractRiveSources(obj)

		for _, src := range sources {

			streams = append(streams, Stream{

				Name:     fmt.Sprintf("Rive [%s]", label),
				URL:      src.url,
				Quality:  src.quality,
				Provider: "Rive",
				Headers:  map[string]string{"Referer": riveBase + "/"},

			})

		}

	}

	return streams, nil

}

type riveSource struct {
	url     string
	quality string
}

func extractRiveSources(obj map[string]any) []riveSource {

	var rawSources []any

	switch {

	case obj["sources"] != nil:

		rawSources, _ = obj["sources"].([]any)

	case obj["data"] != nil:

		data, _ := obj["data"].(map[string]any)

		if data != nil {

			rawSources, _ = data["sources"].([]any)

		}

	}

	var out []riveSource

	for _, s := range rawSources {

		src, _ := s.(map[string]any)

		if src == nil {

			continue

		}

		u, _ := src["url"].(string)

		if u == "" {

			u, _ = src["file"].(string)

		}

		if u == "" {

			u, _ = src["link"].(string)

		}

		if u == "" {

			continue

		}

		quality, _ := src["quality"].(string)

		if quality == "" {

			quality, _ = src["label"].(string)

		}

		if quality == "" {

			quality = "Auto"

		}

		out = append(out, riveSource{url: u, quality: quality})

	}

	return out

}
