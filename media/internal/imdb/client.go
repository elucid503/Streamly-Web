package imdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"mediakit/internal/textutil"
)

var (
	baseURL = envOr("CINEMETA_BASE_URL", "https://v3-cinemeta.strem.io/meta")
	metahubBaseURL = envOr("METAHUB_BASE_URL", "https://episodes.metahub.space")
)

func envOr(key, fallback string) string {

	if v := os.Getenv(key); v != "" {

		return v

	}

	return fallback

}

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

// Client fetches IMDb-indexed metadata from the public Cinemeta catalog.
type Client struct {

	http *http.Client
	mu sync.Mutex

	series map[string]seriesEntry
	movies map[string]movieEntry

}

type seriesEntry struct {

	meta TitleMeta
	episodes map[string]EpisodeMeta

	expiry time.Time

}

type movieEntry struct {

	meta TitleMeta
	expiry time.Time

}

const cacheTTL = 24 * time.Hour

// New builds an IMDb metadata client.
func New() *Client {

	return &Client{

		http: &http.Client{Timeout: 10 * time.Second},

		series: make(map[string]seriesEntry),
		movies: make(map[string]movieEntry),

	}

}

// Series returns metadata for a TV series, caching all episode entries in one request.
func (c *Client) Series(imdbID string) (TitleMeta, error) {

	id := normalizeID(imdbID)

	if id == "" {

		return TitleMeta{}, fmt.Errorf("imdb: missing id")

	}

	c.mu.Lock()

	if entry, ok := c.series[id]; ok && time.Now().Before(entry.expiry) {

		meta := entry.meta
		c.mu.Unlock()

		return meta, nil

	}

	c.mu.Unlock()

	entry, err := c.fetchSeries(id)

	if err != nil {

		return TitleMeta{}, err

	}

	c.mu.Lock()
	c.series[id] = entry
	c.mu.Unlock()

	return entry.meta, nil

}

// Movie returns metadata for a film.
func (c *Client) Movie(imdbID string) (TitleMeta, error) {

	id := normalizeID(imdbID)

	if id == "" {

		return TitleMeta{}, fmt.Errorf("imdb: missing id")

	}

	c.mu.Lock()

	if entry, ok := c.movies[id]; ok && time.Now().Before(entry.expiry) {

		meta := entry.meta
		c.mu.Unlock()

		return meta, nil

	}

	c.mu.Unlock()

	meta, err := c.fetchMovie(id)

	if err != nil {

		return TitleMeta{}, err

	}

	c.mu.Lock()
	c.movies[id] = movieEntry{meta: meta, expiry: time.Now().Add(cacheTTL)}
	c.mu.Unlock()

	return meta, nil

}

// Episode looks up one episode from the cached series payload.
func (c *Client) Episode(imdbID string, season, episode int) (EpisodeMeta, bool) {

	id := normalizeID(imdbID)

	if id == "" || season < 0 || episode <= 0 {

		return EpisodeMeta{}, false

	}

	key := episodeKey(season, episode)

	c.mu.Lock()

	entry, ok := c.series[id]

	if ok && time.Now().Before(entry.expiry) {

		if ep, found := entry.episodes[key]; found {

			c.mu.Unlock()
			return ep, true

		}

	}

	c.mu.Unlock()

	if _, err := c.Series(id); err != nil {

		return EpisodeMeta{}, false

	}

	c.mu.Lock()
	ep, found := c.series[id].episodes[key]
	c.mu.Unlock()

	return ep, found

}

// SeasonEpisodes returns episode metadata for one season.
func (c *Client) SeasonEpisodes(imdbID string, season int) map[int]EpisodeMeta {

	id := normalizeID(imdbID)

	if id == "" || season < 0 {

		return nil

	}

	if _, err := c.Series(id); err != nil {

		return nil

	}

	c.mu.Lock()
	entry := c.series[id]
	c.mu.Unlock()

	out := make(map[int]EpisodeMeta)
	prefix := fmt.Sprintf("%d:", season)

	for key, ep := range entry.episodes {

		if !strings.HasPrefix(key, prefix) {

			continue

		}

		var number int

		if _, err := fmt.Sscanf(key, prefix+"%d", &number); err == nil && number > 0 {

			out[number] = ep

		}

	}

	return out

}

func (c *Client) fetchSeries(imdbID string) (seriesEntry, error) {

	var payload struct {
		Meta cinemetaTitle `json:"meta"`
	}

	if err := c.getJSON(fmt.Sprintf("%s/series/%s.json", baseURL, imdbID), &payload); err != nil {

		return seriesEntry{}, err

	}

	meta := titleFromCinemeta(payload.Meta)
	episodes := make(map[string]EpisodeMeta)

	for _, video := range payload.Meta.Videos {

		if video.Season <= 0 || video.Episode <= 0 {

			continue

		}

		episodes[episodeKey(video.Season, video.Episode)] = EpisodeMeta{

			Title: textutil.DecodeHTML(strings.TrimSpace(video.Name)),
			Description: textutil.DecodeHTML(strings.TrimSpace(firstNonEmpty(video.Description, video.Overview))),
			Poster: firstNonEmpty(video.Thumbnail, episodeStill(imdbID, video.Season, video.Episode)),

		}

	}

	return seriesEntry{

		meta: meta,
		episodes: episodes,
		expiry: time.Now().Add(cacheTTL),

	}, nil

}

func (c *Client) fetchMovie(imdbID string) (TitleMeta, error) {

	var payload struct {
		Meta cinemetaTitle `json:"meta"`
	}

	if err := c.getJSON(fmt.Sprintf("%s/movie/%s.json", baseURL, imdbID), &payload); err != nil {

		return TitleMeta{}, err

	}

	return titleFromCinemeta(payload.Meta), nil

}

type cinemetaTitle struct {

	Name string `json:"name"`
	Year string `json:"year"`

	ReleaseInfo string `json:"releaseInfo"`
	Poster string `json:"poster"`
	Background string `json:"background"`

	Description string `json:"description"`

	IMDBRating string `json:"imdbRating"`
	Videos []cinemetaVideo `json:"videos"`

}

type cinemetaVideo struct {

	Season int `json:"season"`
	Episode int `json:"episode"`
	Number int `json:"number"`

	Name string `json:"name"`
	Description string `json:"description"`
	Overview string `json:"overview"`

	Thumbnail string `json:"thumbnail"`

}

func titleFromCinemeta(raw cinemetaTitle) TitleMeta {

	poster := posterLarge(raw.Poster)

	if poster == "" && raw.Poster != "" {

		poster = raw.Poster

	}

	banner := bannerLarge(raw.Background)

	if banner == "" {

		banner = strings.TrimSpace(raw.Background)

	}

	return TitleMeta{

		Title: textutil.DecodeHTML(strings.TrimSpace(raw.Name)),

		Year: firstNonEmpty(raw.ReleaseInfo, raw.Year),
		Poster: poster,
		Banner: banner,

		Description: textutil.DecodeHTML(strings.TrimSpace(raw.Description)),

		Rating: strings.TrimSpace(raw.IMDBRating),

	}

}

func (c *Client) getJSON(url string, dest any) error {

	resp, err := c.http.Get(url)

	if err != nil {

		return err

	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {

		return err

	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		return fmt.Errorf("imdb: %s → %s", url, resp.Status)

	}

	return json.Unmarshal(body, dest)

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

func episodeKey(season, episode int) string {

	return fmt.Sprintf("%d:%d", season, episode)

}

func episodeStill(imdbID string, season, episode int) string {

	if imdbID == "" || season < 0 || episode <= 0 {

		return ""

	}

	return fmt.Sprintf("%s/%s/%d/%d/w780.jpg", strings.TrimRight(metahubBaseURL, "/"), imdbID, season, episode)

}

func bannerLarge(url string) string {

	url = strings.TrimSpace(url)

	if url == "" {

		return ""

	}

	if strings.Contains(url, "/background/medium/") {

		return strings.Replace(url, "/background/medium/", "/background/large/", 1)

	}

	if strings.Contains(url, "/background/small/") {

		return strings.Replace(url, "/background/small/", "/background/large/", 1)

	}

	return url

}

func posterLarge(url string) string {

	url = strings.TrimSpace(url)

	if url == "" {

		return ""

	}

	if strings.Contains(url, "/poster/small/") {

		return strings.Replace(url, "/poster/small/", "/poster/large/", 1)

	}

	if strings.Contains(url, "/poster/medium/") {

		return strings.Replace(url, "/poster/medium/", "/poster/large/", 1)

	}

	return url

}

func firstNonEmpty(values ...string) string {

	for _, value := range values {

		if strings.TrimSpace(value) != "" {

			return strings.TrimSpace(value)

		}

	}

	return ""

}
