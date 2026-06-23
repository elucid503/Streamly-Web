package captions

import (

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

)

const (
	openSubsBaseURL   = "https://api.opensubtitles.com/api/v1"
	openSubsUserAgent = "Streamly-Web v1.0"
)

// OpenSubsOptions configures the OpenSubtitles client.
type OpenSubsOptions struct {
	APIKey string
}

// OpenSubsClient queries the OpenSubtitles REST API v3 for English subtitle tracks.
type OpenSubsClient struct {

	apiKey string
	http   *http.Client

}

// NewOpenSubsClient builds an OpenSubtitles client.
func NewOpenSubsClient(opts OpenSubsOptions) *OpenSubsClient {

	return &OpenSubsClient{

		apiKey: strings.TrimSpace(opts.APIKey),
		http:   &http.Client{Timeout: 20 * time.Second},

	}

}

// Configured reports whether an API key is present.
func (c *OpenSubsClient) Configured() bool {

	return c.apiKey != ""

}

// ListTracks searches OpenSubtitles for English subtitles matching the query.
func (c *OpenSubsClient) ListTracks(ctx context.Context, query Query) ([]Track, error) {

	if !c.Configured() {

		return nil, ErrUnconfigured

	}

	params := url.Values{}
	params.Set("languages", "en")

	if imdbID := imdbQueryID(query.IMDBId); imdbID != "" {

		params.Set("imdb_id", imdbID)

	} else if query.TMDBId > 0 {

		params.Set("tmdb_id", strconv.Itoa(query.TMDBId))

		if query.Season > 0 && query.Episode > 0 {

			params.Set("type", "episode")

		} else {

			params.Set("type", "movie")

		}

	} else {

		return nil, ErrNoSubtitle

	}

	if query.Season > 0 {

		params.Set("season_number", strconv.Itoa(query.Season))

	}

	if query.Episode > 0 {

		params.Set("episode_number", strconv.Itoa(query.Episode))

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openSubsBaseURL+"/subtitles?"+params.Encode(), nil)

	if err != nil {

		return nil, err

	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {

		return nil, err

	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {

		return nil, fmt.Errorf("%w: %s", ErrUnauthorized, strings.TrimSpace(string(body)))

	}

	if resp.StatusCode == 429 {

		return nil, ErrRateLimited

	}

	if resp.StatusCode >= 400 {

		return nil, fmt.Errorf("captions: opensubtitles returned %d", resp.StatusCode)

	}

	var searchResp openSubsSearchResponse

	if err := json.Unmarshal(body, &searchResp); err != nil {

		return nil, err

	}

	return pickOpenSubsTracks(searchResp.Data), nil

}

// DownloadTrack fetches the subtitle file content for a track returned by ListTracks.
// The track.Path field holds the file_id as a string.
func (c *OpenSubsClient) DownloadTrack(ctx context.Context, track Track, season, episode int) ([]byte, string, error) {

	if !c.Configured() {

		return nil, "", ErrUnconfigured

	}

	fileID, err := strconv.Atoi(strings.TrimSpace(track.Path))

	if err != nil {

		return nil, "", fmt.Errorf("captions: invalid opensubtitles file_id %q: %w", track.Path, err)

	}

	// Step 1: request a download link.
	downloadLink, err := c.requestDownloadLink(ctx, fileID)

	if err != nil {

		return nil, "", err

	}

	// Step 2: download the subtitle content.
	data, err := c.downloadContent(ctx, downloadLink)

	if err != nil {

		return nil, "", err

	}

	// Step 3: extract/normalize (handles ZIP archives just in case).
	content, format, err := extractSubtitle(data, season, episode)

	if err != nil {

		return nil, "", err

	}

	if format == "" {

		format = track.Format

	}

	return content, format, nil

}

func (c *OpenSubsClient) requestDownloadLink(ctx context.Context, fileID int) (string, error) {

	payload := map[string]any{"file_id": fileID}

	body, err := json.Marshal(payload)

	if err != nil {

		return "", err

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openSubsBaseURL+"/download", bytes.NewReader(body))

	if err != nil {

		return "", err

	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)

	if err != nil {

		return "", err

	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {

		return "", err

	}

	if resp.StatusCode == 406 {

		return "", fmt.Errorf("%w: download quota exceeded", ErrRateLimited)

	}

	if resp.StatusCode >= 400 {

		return "", fmt.Errorf("captions: opensubtitles download request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))

	}

	var dlResp openSubsDownloadResponse

	if err := json.Unmarshal(respBody, &dlResp); err != nil {

		return "", err

	}

	if dlResp.Link == "" {

		return "", fmt.Errorf("captions: opensubtitles returned empty download link")

	}

	return dlResp.Link, nil

}

func (c *OpenSubsClient) downloadContent(ctx context.Context, link string) ([]byte, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)

	if err != nil {

		return nil, err

	}

	req.Header.Set("User-Agent", openSubsUserAgent)

	resp, err := c.http.Do(req)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {

		return nil, fmt.Errorf("captions: subtitle download returned %d", resp.StatusCode)

	}

	return io.ReadAll(resp.Body)

}

func (c *OpenSubsClient) setHeaders(req *http.Request) {

	req.Header.Set("Api-Key", c.apiKey)
	req.Header.Set("User-Agent", openSubsUserAgent)
	req.Header.Set("Accept", "application/json")

}

// --- Response types ---

type openSubsSearchResponse struct {

	Data []openSubsItem `json:"data"`

}

type openSubsItem struct {

	Attributes openSubsAttributes `json:"attributes"`

}

type openSubsAttributes struct {

	Language        string        `json:"language"`
	HearingImpaired bool          `json:"hearing_impaired"`
	Files           []openSubsFile `json:"files"`

}

type openSubsFile struct {

	FileID   int    `json:"file_id"`
	FileName string `json:"file_name"`

}

type openSubsDownloadResponse struct {

	Link     string `json:"link"`
	FileName string `json:"file_name"`

}

// --- helpers ---

func pickOpenSubsTracks(items []openSubsItem) []Track {

	seen := make(map[int]struct{})
	var tracks []Track

	for _, item := range items {

		if !looksEnglishLanguageTag(item.Attributes.Language) {

			continue

		}

		for _, f := range item.Attributes.Files {

			if f.FileID <= 0 {

				continue

			}

			if _, dup := seen[f.FileID]; dup {

				continue

			}

			seen[f.FileID] = struct{}{}

			format := normalizeFormat("", f.FileName)

			if format == "zip" {

				continue

			}

			tracks = append(tracks, Track{

				Path:     strconv.Itoa(f.FileID),
				Name:     f.FileName,
				Language: "en",
				Format:   format,
				Hi:       item.Attributes.HearingImpaired,

			})

		}

	}

	return tracks

}
