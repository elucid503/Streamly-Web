package providers

import (

	"encoding/json"
	"fmt"

)

const vidlinkBase = "https://vidlink.pro"

type vidlinkProvider struct{}

func newVidlink() Provider { return &vidlinkProvider{} }

func (v *vidlinkProvider) Name() string { return "Vidlink" }

func (v *vidlinkProvider) Fetch(tmdbID int, mediaType string, season, episode int) ([]Stream, error) {

	// Step 1: Encrypt TMDB ID via enc-dec.app
	encURL := fmt.Sprintf("https://enc-dec.app/api/enc-vidlink?text=%d", tmdbID)

	encData, err := getJSON(encURL, nil)

	if err != nil {

		return nil, fmt.Errorf("vidlink: encryption step failed: %w", err)

	}

	encodedTmdb, _ := encData["result"].(string)

	if encodedTmdb == "" {

		return nil, fmt.Errorf("vidlink: encryption returned empty result")

	}

	// Step 2: Fetch stream from Vidlink API
	var apiURL string

	if mediaType == "tv" || mediaType == "series" {

		apiURL = fmt.Sprintf("%s/api/b/tv/%s/%d/%d?multiLang=0", vidlinkBase, encodedTmdb, season, episode)

	} else {

		apiURL = fmt.Sprintf("%s/api/b/movie/%s?multiLang=0", vidlinkBase, encodedTmdb)

	}

	raw, err := getJSONRaw(apiURL, map[string]string{"Referer": vidlinkBase})

	if err != nil {

		return nil, fmt.Errorf("vidlink: api call failed: %w", err)

	}

	var resp map[string]any

	if err := json.Unmarshal(raw, &resp); err != nil {

		return nil, fmt.Errorf("vidlink: invalid json: %w", err)

	}

	streamObj, _ := resp["stream"].(map[string]any)

	if streamObj == nil {

		return nil, fmt.Errorf("vidlink: no stream object in response")

	}

	playlist, _ := streamObj["playlist"].(string)

	if playlist == "" {

		return nil, fmt.Errorf("vidlink: no playlist URL in response")

	}

	return []Stream{

		{

			Name:     "Vidlink",
			URL:      playlist,
			Quality:  "Auto",
			Provider: "Vidlink",
			Headers:  map[string]string{"Referer": vidlinkBase},

		},

	}, nil

}
