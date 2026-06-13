package captions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	subDLBaseURL   = "https://api.subdl.com/api/v1"
	subDLDownload  = "https://dl.subdl.com"
	subDLUserAgent = "Streamly-Web/1.0"
)

var (
	episodeTagRE     = regexp.MustCompile(`(?i)(?:^|[.\s_-])s(\d{1,2})e(\d{1,2})(?:[.\s_-]|$)`)
	episodeXRE       = regexp.MustCompile(`(?i)(?:^|[.\s_-])(\d{1,2})x(\d{1,2})(?:[.\s_-]|$)`)
	leadingEpisodeRE = regexp.MustCompile(`(?i)^(\d{1,2})\s+`)
)

type SubDLOptions struct {
	APIKey string
}

type SubDLClient struct {
	apiKey string
	http   *http.Client
}

type subdlUnpackFile struct {
	URL         string `json:"url"`
	Name        string `json:"name"`
	ReleaseName string `json:"release_name"`
	Season      int    `json:"season"`
	Episode     int    `json:"episode"`
	Format      string `json:"format"`
	Language    string `json:"language"`
	Hi          bool   `json:"hi"`
}

type subdlSubtitle struct {
	ReleaseName string            `json:"release_name"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Season      int               `json:"season"`
	Episode     int               `json:"episode"`
	Hi          bool              `json:"hi"`
	UnpackFiles []subdlUnpackFile `json:"unpack_files"`
}

type subdlSearchResponse struct {
	Status    bool            `json:"status"`
	Error     string          `json:"error"`
	Subtitles []subdlSubtitle `json:"subtitles"`
}

func NewSubDLClient(options SubDLOptions) *SubDLClient {
	return &SubDLClient{
		apiKey: strings.TrimSpace(options.APIKey),
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *SubDLClient) Configured() bool {
	return c.apiKey != ""
}

func (c *SubDLClient) ListTracks(ctx context.Context, query Query) ([]Track, error) {
	if !c.Configured() {
		return nil, ErrUnconfigured
	}

	response, err := c.search(ctx, query)
	if err != nil {
		return nil, err
	}

	tracks := pickSubDLTracks(response, query.Season, query.Episode)
	if len(tracks) == 0 {
		return nil, ErrNoSubtitle
	}
	return tracks, nil
}

func (c *SubDLClient) DownloadTrack(ctx context.Context, track Track, season, episode int) ([]byte, string, error) {
	data, err := c.downloadBytes(ctx, track.Path)
	if err != nil {
		return nil, "", err
	}
	return extractSubtitle(data, season, episode)
}

func (c *SubDLClient) search(ctx context.Context, query Query) (subdlSearchResponse, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("languages", "EN")
	params.Set("unpack", "1")

	if query.Season > 0 && query.Episode > 0 {
		params.Set("type", "tv")
		params.Set("season_number", strconv.Itoa(query.Season))
		params.Set("episode_number", strconv.Itoa(query.Episode))
	} else {
		params.Set("type", "movie")
	}

	if imdb := imdbQueryID(query.IMDBId); imdb != "" {
		params.Set("imdb_id", imdb)
	} else if query.TMDBId > 0 {
		params.Set("tmdb_id", strconv.Itoa(query.TMDBId))
	} else {
		return subdlSearchResponse{}, ErrNoSubtitle
	}

	if name := strings.TrimSpace(query.VideoName); name != "" {
		params.Set("file_name", name)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, subDLBaseURL+"/subtitles?"+params.Encode(), nil)
	if err != nil {
		return subdlSearchResponse{}, err
	}
	req.Header.Set("User-Agent", subDLUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return subdlSearchResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return subdlSearchResponse{}, err
	}

	if resp.StatusCode >= 400 {
		return subdlSearchResponse{}, mapSubDLError(resp.StatusCode, string(body))
	}

	var response subdlSearchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return subdlSearchResponse{}, err
	}

	if !response.Status {
		if strings.TrimSpace(response.Error) != "" {
			return subdlSearchResponse{}, fmt.Errorf("captions: subdl: %s", strings.TrimSpace(response.Error))
		}
		return subdlSearchResponse{}, ErrNoSubtitle
	}

	return response, nil
}

func pickSubDLTracks(response subdlSearchResponse, season, episode int) []Track {
	seen := make(map[string]struct{})
	var tracks []Track

	add := func(path, name, language, format string, hi bool) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		tracks = append(tracks, Track{
			Path:     path,
			Name:     strings.TrimSpace(name),
			Language: normalizeLanguage(language),
			Format:   normalizeFormat(format, name),
			Hi:       hi,
		})
	}

	if season > 0 && episode > 0 {
		for _, subtitle := range response.Subtitles {
			for _, file := range subtitle.UnpackFiles {
				if !fileMatchesEpisode(file, season, episode) {
					continue
				}
				add(file.URL, file.Name, file.Language, file.Format, file.Hi)
			}
		}
		for _, subtitle := range response.Subtitles {
			if !subtitleMatchesEpisode(subtitle, season, episode) {
				continue
			}
			if len(subtitle.UnpackFiles) == 1 {
				file := subtitle.UnpackFiles[0]
				if looksEnglishLanguageTag(file.Language) && !hasForeignLanguageName(file.Name) {
					add(file.URL, file.Name, file.Language, file.Format, file.Hi)
				}
			}
			if len(subtitle.UnpackFiles) == 0 {
				path := strings.TrimSpace(subtitle.URL)
				if path != "" && !strings.HasSuffix(strings.ToLower(path), ".zip") {
					add(path, subtitle.Name, "en", "", subtitle.Hi)
				}
			}
		}
		for _, path := range pickSubDLSeasonZipPaths(response, season) {
			add(path, subtitleZipLabel(path), "en", "zip", false)
		}
		return tracks
	}

	for _, subtitle := range response.Subtitles {
		for _, file := range subtitle.UnpackFiles {
			if !looksEnglishLanguageTag(file.Language) || hasForeignLanguageName(file.Name) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(file.Format), "srt") || strings.HasSuffix(strings.ToLower(file.Name), ".srt") || strings.HasSuffix(strings.ToLower(file.Name), ".vtt") {
				add(file.URL, file.Name, file.Language, file.Format, file.Hi)
			}
		}
		if path := strings.TrimSpace(subtitle.URL); path != "" {
			add(path, subtitle.Name, "en", "", subtitle.Hi)
		}
	}

	return tracks
}

func subtitleZipLabel(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "English"
	}
	return parts[len(parts)-1]
}

func normalizeLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	switch language {
	case "en", "eng", "english", "en-us", "en-gb", "en_us", "en_gb", "":
		return "en"
	default:
		return language
	}
}

func normalizeFormat(format, name string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "srt" || format == "vtt" {
		return format
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".vtt") {
		return "vtt"
	}
	if strings.HasSuffix(lower, ".zip") {
		return "zip"
	}
	return "srt"
}

func pickSubDLSeasonZipPaths(response subdlSearchResponse, season int) []string {
	var preferred []string
	var fallback []string
	for _, subtitle := range response.Subtitles {
		if !seasonMatches(subtitle.Season, season) {
			continue
		}
		path := strings.TrimSpace(subtitle.URL)
		if path == "" || !strings.HasSuffix(strings.ToLower(path), ".zip") {
			continue
		}
		joined := strings.ToLower(subtitle.ReleaseName + " " + subtitle.Name)
		if strings.Contains(joined, "forced") {
			fallback = append(fallback, path)
			continue
		}
		preferred = append(preferred, path)
	}
	return append(preferred, fallback...)
}

func nameMatchesEpisode(name string, season, episode int) bool {
	if s, e := parseEpisodeTag(name); e == episode && seasonMatches(s, season) {
		return true
	}
	if e := parseLeadingEpisode(name); e == episode {
		return true
	}
	return false
}

func subtitleMatchesEpisode(subtitle subdlSubtitle, season, episode int) bool {
	if subtitle.Episode == episode && seasonMatches(subtitle.Season, season) {
		return true
	}
	for _, label := range []string{subtitle.ReleaseName, subtitle.Name} {
		if s, e := parseEpisodeTag(label); e == episode && seasonMatches(s, season) {
			return true
		}
	}
	return false
}

func fileMatchesEpisode(file subdlUnpackFile, season, episode int) bool {
	if !looksEnglishLanguageTag(file.Language) || hasForeignLanguageName(file.Name) {
		return false
	}
	for _, label := range []string{file.Name, file.ReleaseName} {
		if s, e := parseEpisodeTag(label); e == episode && seasonMatches(s, season) {
			return true
		}
		if e := parseLeadingEpisode(label); e == episode {
			return true
		}
	}
	if file.Episode == episode && seasonMatches(file.Season, season) {
		return true
	}
	return false
}

func seasonMatches(got, want int) bool {
	return got == 0 || got == want
}

func parseEpisodeTag(label string) (season, episode int) {
	label = strings.TrimSpace(label)
	if label == "" {
		return 0, 0
	}
	if match := episodeTagRE.FindStringSubmatch(label); len(match) == 3 {
		season, _ = strconv.Atoi(match[1])
		episode, _ = strconv.Atoi(match[2])
		return season, episode
	}
	if match := episodeXRE.FindStringSubmatch(label); len(match) == 3 {
		season, _ = strconv.Atoi(match[1])
		episode, _ = strconv.Atoi(match[2])
		return season, episode
	}
	return 0, 0
}

func parseLeadingEpisode(label string) int {
	label = strings.TrimSpace(label)
	if label == "" {
		return 0
	}
	base := label
	if idx := strings.Index(label, "/"); idx >= 0 {
		base = label[idx+1:]
	}
	match := leadingEpisodeRE.FindStringSubmatch(base)
	if len(match) != 2 {
		return 0
	}
	episode, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return episode
}

func (c *SubDLClient) downloadBytes(ctx context.Context, path string) ([]byte, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	downloadURL := subDLDownload + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", subDLUserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, mapSubDLError(resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func imdbQueryID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(id), "tt") {
		return "tt" + id
	}
	return id
}

func mapSubDLError(status int, body string) error {
	switch status {
	case 401, 403:
		return fmt.Errorf("%w: %s", ErrUnauthorized, strings.TrimSpace(body))
	case 429:
		return fmt.Errorf("%w: %s", ErrRateLimited, strings.TrimSpace(body))
	case 404:
		return ErrNoSubtitle
	default:
		return fmt.Errorf("captions: subdl request failed with status %d", status)
	}
}