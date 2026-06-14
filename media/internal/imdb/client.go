package imdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	tmdbBaseURL   = "https://api.themoviedb.org/3"
	tmdbImageBase = "https://image.tmdb.org/t/p"
	cacheTTL      = 24 * time.Hour
)

// TitleMeta is display metadata keyed by an IMDb id.
type TitleMeta struct {

	Title string
	Year string

	Poster string
	Banner string

	Description string

	Rating string

}

// EpisodeMeta is per-episode metadata for a TV series.
type EpisodeMeta struct {

	Title string
	Description string

	Poster string

}

// Client fetches metadata from the TMDB API using an IMDb ID as the lookup key.
type Client struct {

	apiKey string
	http   *http.Client

	mu sync.Mutex

	idMap  map[string]int         // imdbID → tmdbID
	series map[string]seriesEntry // imdbID → series metadata
	movies map[string]movieEntry  // imdbID → movie metadata
	seasons map[string]seasonEntry // "imdbID:season" → episodes

}

type seriesEntry struct {
	meta   TitleMeta
	expiry time.Time
}

type movieEntry struct {
	meta   TitleMeta
	expiry time.Time
}

type seasonEntry struct {
	episodes map[int]EpisodeMeta
	expiry   time.Time
}

// New builds a TMDB metadata client. apiKey is the TMDB API read access token
// (v3 auth key). If empty, all enrichment calls are skipped gracefully.
func New(apiKey string) *Client {

	return &Client{

		apiKey:  apiKey,
		http:    &http.Client{Timeout: 8 * time.Second},
		idMap:   make(map[string]int),
		series:  make(map[string]seriesEntry),
		movies:  make(map[string]movieEntry),
		seasons: make(map[string]seasonEntry),

	}

}

// Series returns metadata for a TV series by IMDb ID.
func (c *Client) Series(imdbID string) (TitleMeta, error) {

	if c.apiKey == "" {

		return TitleMeta{}, fmt.Errorf("tmdb: no api key")

	}

	id := normalizeID(imdbID)

	if id == "" {

		return TitleMeta{}, fmt.Errorf("tmdb: missing id")

	}

	c.mu.Lock()

	if entry, ok := c.series[id]; ok && time.Now().Before(entry.expiry) {

		meta := entry.meta
		c.mu.Unlock()
		return meta, nil

	}

	c.mu.Unlock()

	tmdbID, err := c.tmdbID(id, "tv")

	if err != nil {

		return TitleMeta{}, err

	}

	var raw tmdbTV

	if err := c.getJSON(fmt.Sprintf("%s/tv/%d", tmdbBaseURL, tmdbID), &raw); err != nil {

		return TitleMeta{}, err

	}

	meta := metaFromTV(raw)

	c.mu.Lock()
	c.series[id] = seriesEntry{meta: meta, expiry: time.Now().Add(cacheTTL)}
	c.mu.Unlock()

	return meta, nil

}

// Movie returns metadata for a film by IMDb ID.
func (c *Client) Movie(imdbID string) (TitleMeta, error) {

	if c.apiKey == "" {

		return TitleMeta{}, fmt.Errorf("tmdb: no api key")

	}

	id := normalizeID(imdbID)

	if id == "" {

		return TitleMeta{}, fmt.Errorf("tmdb: missing id")

	}

	c.mu.Lock()

	if entry, ok := c.movies[id]; ok && time.Now().Before(entry.expiry) {

		meta := entry.meta
		c.mu.Unlock()
		return meta, nil

	}

	c.mu.Unlock()

	tmdbID, err := c.tmdbID(id, "movie")

	if err != nil {

		return TitleMeta{}, err

	}

	var raw tmdbMovie

	if err := c.getJSON(fmt.Sprintf("%s/movie/%d", tmdbBaseURL, tmdbID), &raw); err != nil {

		return TitleMeta{}, err

	}

	meta := metaFromMovie(raw)

	c.mu.Lock()
	c.movies[id] = movieEntry{meta: meta, expiry: time.Now().Add(cacheTTL)}
	c.mu.Unlock()

	return meta, nil

}

// Episode returns metadata for a single episode by IMDb ID, season, and episode number.
func (c *Client) Episode(imdbID string, season, episode int) (EpisodeMeta, bool) {

	eps := c.SeasonEpisodes(imdbID, season)

	ep, ok := eps[episode]

	return ep, ok

}

// SeasonEpisodes returns all episode metadata for one season, keyed by episode number.
func (c *Client) SeasonEpisodes(imdbID string, season int) map[int]EpisodeMeta {

	if c.apiKey == "" || season < 0 {

		return nil

	}

	id := normalizeID(imdbID)

	if id == "" {

		return nil

	}

	cacheKey := fmt.Sprintf("%s:%d", id, season)

	c.mu.Lock()

	if entry, ok := c.seasons[cacheKey]; ok && time.Now().Before(entry.expiry) {

		out := make(map[int]EpisodeMeta, len(entry.episodes))
		for k, v := range entry.episodes {
			out[k] = v
		}
		c.mu.Unlock()
		return out

	}

	c.mu.Unlock()

	tmdbID, err := c.tmdbID(id, "tv")

	if err != nil {

		return nil

	}

	var raw tmdbSeason

	if err := c.getJSON(fmt.Sprintf("%s/tv/%d/season/%d", tmdbBaseURL, tmdbID, season), &raw); err != nil {

		return nil

	}

	eps := make(map[int]EpisodeMeta, len(raw.Episodes))

	for _, ep := range raw.Episodes {

		if ep.EpisodeNumber <= 0 {

			continue

		}

		eps[ep.EpisodeNumber] = EpisodeMeta{

			Title:       strings.TrimSpace(ep.Name),
			Description: strings.TrimSpace(ep.Overview),
			Poster:      imagePath(ep.StillPath, "w300"),

		}

	}

	c.mu.Lock()
	c.seasons[cacheKey] = seasonEntry{episodes: eps, expiry: time.Now().Add(cacheTTL)}
	c.mu.Unlock()

	return eps

}

// tmdbID resolves the TMDB integer ID for an IMDb ID, caching the result.
// mediaType is "tv" or "movie".
func (c *Client) tmdbID(imdbID, mediaType string) (int, error) {

	c.mu.Lock()

	if id, ok := c.idMap[imdbID]; ok {

		c.mu.Unlock()
		return id, nil

	}

	c.mu.Unlock()

	var result tmdbFindResponse

	url := fmt.Sprintf("%s/find/%s?external_source=imdb_id", tmdbBaseURL, imdbID)

	if err := c.getJSON(url, &result); err != nil {

		return 0, err

	}

	var tmdbID int

	if mediaType == "tv" && len(result.TVResults) > 0 {

		tmdbID = result.TVResults[0].ID

	} else if mediaType == "movie" && len(result.MovieResults) > 0 {

		tmdbID = result.MovieResults[0].ID

	}

	if tmdbID == 0 {

		return 0, fmt.Errorf("tmdb: no %s result for %s", mediaType, imdbID)

	}

	c.mu.Lock()
	c.idMap[imdbID] = tmdbID
	c.mu.Unlock()

	return tmdbID, nil

}

func (c *Client) getJSON(url string, dest any) error {

	reqURL := url

	// TMDB v3 hex keys use ?api_key=; JWT Read Access Tokens use Authorization: Bearer.
	if !strings.HasPrefix(c.apiKey, "eyJ") {

		sep := "?"

		if strings.Contains(url, "?") {

			sep = "&"

		}

		reqURL = url + sep + "api_key=" + c.apiKey

	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)

	if err != nil {

		return err

	}

	if strings.HasPrefix(c.apiKey, "eyJ") {

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)

	if err != nil {

		return err

	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {

		return err

	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		return fmt.Errorf("tmdb: %s → %s", url, resp.Status)

	}

	return json.Unmarshal(body, dest)

}

// --- TMDB response types ---

type tmdbFindResponse struct {

	TVResults []struct {
		ID int `json:"id"`
	} `json:"tv_results"`

	MovieResults []struct {
		ID int `json:"id"`
	} `json:"movie_results"`

}

type tmdbTV struct {

	Name         string  `json:"name"`
	FirstAirDate string  `json:"first_air_date"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	Overview     string  `json:"overview"`
	VoteAverage  float64 `json:"vote_average"`

}

type tmdbMovie struct {

	Title        string  `json:"title"`
	ReleaseDate  string  `json:"release_date"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	Overview     string  `json:"overview"`
	VoteAverage  float64 `json:"vote_average"`

}

type tmdbSeason struct {

	Episodes []tmdbEpisode `json:"episodes"`

}

type tmdbEpisode struct {

	EpisodeNumber int    `json:"episode_number"`
	Name          string `json:"name"`
	Overview      string `json:"overview"`
	StillPath     string `json:"still_path"`

}

// --- helpers ---

func metaFromTV(raw tmdbTV) TitleMeta {

	year := ""

	if len(raw.FirstAirDate) >= 4 {

		year = raw.FirstAirDate[:4]

	}

	return TitleMeta{

		Title:       strings.TrimSpace(raw.Name),
		Year:        year,
		Poster:      imagePath(raw.PosterPath, "w500"),
		Banner:      imagePath(raw.BackdropPath, "original"),
		Description: strings.TrimSpace(raw.Overview),
		Rating:      formatRating(raw.VoteAverage),

	}

}

func metaFromMovie(raw tmdbMovie) TitleMeta {

	year := ""

	if len(raw.ReleaseDate) >= 4 {

		year = raw.ReleaseDate[:4]

	}

	return TitleMeta{

		Title:       strings.TrimSpace(raw.Title),
		Year:        year,
		Poster:      imagePath(raw.PosterPath, "w500"),
		Banner:      imagePath(raw.BackdropPath, "original"),
		Description: strings.TrimSpace(raw.Overview),
		Rating:      formatRating(raw.VoteAverage),

	}

}

func imagePath(path, size string) string {

	if strings.TrimSpace(path) == "" {

		return ""

	}

	return fmt.Sprintf("%s/%s%s", tmdbImageBase, size, path)

}

func formatRating(v float64) string {

	if v == 0 {

		return ""

	}

	return fmt.Sprintf("%.1f", v)

}

func normalizeID(id string) string {

	id = strings.TrimSpace(id)

	if id == "" {

		return ""

	}

	if strings.HasPrefix(strings.ToLower(id), "tt") {

		return id

	}

	return "tt" + id

}
