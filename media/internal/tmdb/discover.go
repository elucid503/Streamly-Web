package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseURL   = "https://api.themoviedb.org/3"
	imageBase = "https://image.tmdb.org/t/p"
	listTTL   = 2 * time.Hour
)

// MediaKind is movie or tv for TMDB list endpoints.
type MediaKind string

const (

	KindMovie MediaKind = "movie"
	KindTV MediaKind = "tv"

)

// Item is a TMDB list/discover result with fields needed for home feeds.
type Item struct {

	ID int
	Kind MediaKind

	Title string
	Year int

	Poster string
	Backdrop string
	Overview string

	VoteAverage float64
	GenreIDs []int
	Runtime int

}

// Client fetches TMDB discover/trending/recommendation lists.
type Client struct {

	apiKey string
	http *http.Client

	mu sync.Mutex
	cache map[string]cacheEntry
	imdbIDs map[string]string

}

type cacheEntry struct {

	items []Item
	expiry time.Time

}

// New builds a TMDB discover client. Empty apiKey disables all calls gracefully.
func New(apiKey string) *Client {

	return &Client{

		apiKey: strings.TrimSpace(apiKey),
		http: &http.Client{Timeout: 10 * time.Second},
		cache: make(map[string]cacheEntry),
		imdbIDs: make(map[string]string),

	}

}

// Enabled reports whether an API key is configured.
func (c *Client) Enabled() bool {

	return c != nil && c.apiKey != ""

}

// Trending returns weekly trending titles for the given kind.
func (c *Client) Trending(kind MediaKind, limit int) ([]Item, error) {

	path := fmt.Sprintf("/trending/%s/week", kind)

	return c.list(path, nil, kind, limit)

}

// Popular returns popular titles.
func (c *Client) Popular(kind MediaKind, limit int) ([]Item, error) {

	path := fmt.Sprintf("/%s/popular", kind)

	return c.list(path, nil, kind, limit)

}

// NowPlaying returns movies currently in theaters.
func (c *Client) NowPlaying(limit int) ([]Item, error) {

	return c.list("/movie/now_playing", nil, KindMovie, limit)

}

// Upcoming returns upcoming movies.
func (c *Client) Upcoming(limit int) ([]Item, error) {

	return c.list("/movie/upcoming", nil, KindMovie, limit)

}

// OnTheAir returns TV shows currently airing.
func (c *Client) OnTheAir(limit int) ([]Item, error) {

	return c.list("/tv/on_the_air", nil, KindTV, limit)

}

// AiringToday returns TV shows airing today.
func (c *Client) AiringToday(limit int) ([]Item, error) {

	return c.list("/tv/airing_today", nil, KindTV, limit)

}

// Discover returns titles matching genre and vote filters.
func (c *Client) Discover(kind MediaKind, genreIDs []int, minVote float64, minVotes int, limit int) ([]Item, error) {

	params := url.Values{}

	params.Set("sort_by", "popularity.desc")
	params.Set("include_adult", "false")

	if len(genreIDs) > 0 {

		ids := make([]string, len(genreIDs))

		for i, id := range genreIDs {

			ids[i] = strconv.Itoa(id)

		}

		params.Set("with_genres", strings.Join(ids, ","))

	}

	if minVote > 0 {

		params.Set("vote_average.gte", strconv.FormatFloat(minVote, 'f', 1, 64))

	}

	if minVotes > 0 {

		params.Set("vote_count.gte", strconv.Itoa(minVotes))

	}

	path := fmt.Sprintf("/discover/%s", kind)

	return c.list(path, params, kind, limit)

}

// Recommendations returns TMDB recommendations for a title.
func (c *Client) Recommendations(kind MediaKind, tmdbID, limit int) ([]Item, error) {

	if tmdbID <= 0 {

		return nil, fmt.Errorf("tmdb: invalid id")

	}

	path := fmt.Sprintf("/%s/%d/recommendations", kind, tmdbID)

	return c.list(path, nil, kind, limit)

}

// Similar returns similar titles.
func (c *Client) Similar(kind MediaKind, tmdbID, limit int) ([]Item, error) {

	if tmdbID <= 0 {

		return nil, fmt.Errorf("tmdb: invalid id")

	}

	path := fmt.Sprintf("/%s/%d/similar", kind, tmdbID)

	return c.list(path, nil, kind, limit)

}

// Details fetches a single title's genres and runtime (lightweight detail call).
func (c *Client) Details(kind MediaKind, tmdbID int) (Item, error) {

	if !c.Enabled() {

		return Item{}, fmt.Errorf("tmdb: no api key")

	}

	if tmdbID <= 0 {

		return Item{}, fmt.Errorf("tmdb: invalid id")

	}

	var raw map[string]any

	path := fmt.Sprintf("/%s/%d", kind, tmdbID)

	if err := c.getJSON(path, nil, &raw); err != nil {

		return Item{}, err

	}

	return itemFromMap(raw, kind), nil

}

// ExternalIMDB returns the IMDb id for a TMDB title (e.g. "tt1234567").
func (c *Client) ExternalIMDB(kind MediaKind, tmdbID int) (string, error) {

	if !c.Enabled() {

		return "", fmt.Errorf("tmdb: no api key")

	}

	if tmdbID <= 0 {

		return "", fmt.Errorf("tmdb: invalid id")

	}

	cacheKey := string(kind) + ":" + strconv.Itoa(tmdbID)

	c.mu.Lock()

	if id, ok := c.imdbIDs[cacheKey]; ok {

		c.mu.Unlock()
		return id, nil

	}

	c.mu.Unlock()

	var raw struct {

		IMDBID string `json:"imdb_id"`

	}

	path := fmt.Sprintf("/%s/%d/external_ids", kind, tmdbID)

	if err := c.getJSON(path, nil, &raw); err != nil {

		return "", err

	}

	id := strings.TrimSpace(raw.IMDBID)

	c.mu.Lock()
	c.imdbIDs[cacheKey] = id
	c.mu.Unlock()

	return id, nil

}

// Search finds titles by name. Used to resolve watch-history seeds without a Showbox TMDB id.
func (c *Client) Search(kind MediaKind, query string, year, limit int) ([]Item, error) {

	if !c.Enabled() {

		return nil, fmt.Errorf("tmdb: no api key")

	}

	query = strings.TrimSpace(query)

	if query == "" {

		return nil, fmt.Errorf("tmdb: empty query")

	}

	params := url.Values{}

	params.Set("query", query)
	params.Set("include_adult", "false")

	if year > 0 {

		if kind == KindMovie {

			params.Set("year", strconv.Itoa(year))

		} else {

			params.Set("first_air_date_year", strconv.Itoa(year))

		}

	}

	path := fmt.Sprintf("/search/%s", kind)

	return c.list(path, params, kind, limit)

}

func (c *Client) list(path string, params url.Values, kind MediaKind, limit int) ([]Item, error) {

	if !c.Enabled() {

		return nil, fmt.Errorf("tmdb: no api key")

	}

	if limit <= 0 {

		limit = 20

	}

	cacheKey := path + "?" + params.Encode()

	c.mu.Lock()

	if entry, ok := c.cache[cacheKey]; ok && time.Now().Before(entry.expiry) {

		out := append([]Item(nil), entry.items...)
		c.mu.Unlock()

		if len(out) > limit {

			out = out[:limit]

		}

		return out, nil

	}

	c.mu.Unlock()

	var page listPage

	if err := c.getJSON(path, params, &page); err != nil {

		return nil, err

	}

	items := make([]Item, 0, len(page.Results))

	for _, raw := range page.Results {

		items = append(items, itemFromMap(raw, kind))

	}

	c.mu.Lock()
	c.cache[cacheKey] = cacheEntry{items: items, expiry: time.Now().Add(listTTL)}
	c.mu.Unlock()

	if len(items) > limit {

		items = items[:limit]

	}

	return items, nil

}

type listPage struct {

	Results []map[string]any `json:"results"`

}

func (c *Client) getJSON(path string, params url.Values, dest any) error {

	if params == nil {

		params = url.Values{}

	}

	reqURL := baseURL + path

	if !strings.HasPrefix(c.apiKey, "eyJ") {

		params.Set("api_key", c.apiKey)

	}

	if encoded := params.Encode(); encoded != "" {

		reqURL += "?" + encoded

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

	if resp.StatusCode >= 400 {

		return fmt.Errorf("tmdb: %s (%d)", strings.TrimSpace(string(body)), resp.StatusCode)

	}

	return json.Unmarshal(body, dest)

}

func itemFromMap(raw map[string]any, kind MediaKind) Item {

	title := stringField(raw, "title")

	if title == "" {

		title = stringField(raw, "name")

	}

	date := stringField(raw, "release_date")

	if date == "" {

		date = stringField(raw, "first_air_date")

	}

	year := 0

	if len(date) >= 4 {

		year, _ = strconv.Atoi(date[:4])

	}

	genreIDs := intSliceField(raw, "genre_ids")

	if len(genreIDs) == 0 {

		if genres, ok := raw["genres"].([]any); ok {

			for _, g := range genres {

				if m, ok := g.(map[string]any); ok {

					genreIDs = append(genreIDs, intField(m, "id"))

				}

			}

		}

	}

	poster := stringField(raw, "poster_path")
	backdrop := stringField(raw, "backdrop_path")

	return Item{

		ID: intField(raw, "id"),
		Kind: kind,

		Title: title,
		Year: year,

		Poster: imageURL(poster, "w500"),
		Backdrop: imageURL(backdrop, "w1280"),
		Overview: stringField(raw, "overview"),

		VoteAverage: floatField(raw, "vote_average"),
		GenreIDs: genreIDs,
		Runtime: intField(raw, "runtime"),

	}

}

func imageURL(path, size string) string {

	path = strings.TrimSpace(path)

	if path == "" {

		return ""

	}

	if strings.HasPrefix(path, "http") {

		return path

	}

	return imageBase + "/" + size + path

}

func stringField(m map[string]any, key string) string {

	v, ok := m[key]

	if !ok || v == nil {

		return ""

	}

	return strings.TrimSpace(fmt.Sprint(v))

}

func intField(m map[string]any, key string) int {

	v, ok := m[key]

	if !ok || v == nil {

		return 0

	}

	switch n := v.(type) {

	case float64:

		return int(n)

	case int:

		return n

	case json.Number:

		i, _ := n.Int64()
		return int(i)

	default:

		i, _ := strconv.Atoi(fmt.Sprint(v))
		return i

	}

}

func floatField(m map[string]any, key string) float64 {

	v, ok := m[key]

	if !ok || v == nil {

		return 0

	}

	switch n := v.(type) {

	case float64:

		return n

	case int:

		return float64(n)

	default:

		f, _ := strconv.ParseFloat(fmt.Sprint(v), 64)
		return f

	}

}

func intSliceField(m map[string]any, key string) []int {

	v, ok := m[key]

	if !ok || v == nil {

		return nil

	}

	arr, ok := v.([]any)

	if !ok {

		return nil

	}

	out := make([]int, 0, len(arr))

	for _, item := range arr {

		switch n := item.(type) {

		case float64:

			out = append(out, int(n))

		case int:

			out = append(out, n)

		}

	}

	return out

}

// GenreName returns a display name for a TMDB genre id.
func GenreName(id int) string {

	if name, ok := genreNames[id]; ok {

		return name

	}

	return ""

}

// GenreNames maps genre ids to display names, skipping unknowns.
func GenreNames(ids []int) []string {

	out := make([]string, 0, len(ids))

	for _, id := range ids {

		if name := GenreName(id); name != "" {

			out = append(out, name)

		}

	}

	return out

}

var genreNames = map[int]string{

	28: "Action",
	12: "Adventure",
	16: "Animation",
	35: "Comedy",
	80: "Crime",
	99: "Documentary",
	18: "Drama",
	10751: "Family",
	14: "Fantasy",
	36: "History",
	27: "Horror",
	10402: "Music",
	9648: "Mystery",
	10749: "Romance",
	878: "Sci-Fi",
	10770: "TV Movie",
	53: "Thriller",
	10752: "War",
	37: "Western",
	10759: "Action & Adventure",
	10762: "Kids",
	10763: "News",
	10764: "Reality",
	10765: "Sci-Fi & Fantasy",
	10766: "Soap",
	10767: "Talk",
	10768: "War & Politics",

}
