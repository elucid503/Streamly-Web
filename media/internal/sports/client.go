package sports

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"mediakit/internal/tv"
)

const (
	defaultBaseURL = "https://ntv.cx"
	defaultServer  = "kobra"

	matchesTTL = 2 * time.Minute

	browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"
)

// Client fetches sports matches from ntv.cx and matches them to 24/7 channels.
type Client struct {

	baseURL string
	server  string

	httpClient *http.Client
	tvClient   *tv.Client

	mu        sync.RWMutex
	matches   []Match
	matchesAt time.Time

}

// New builds a sports Client. tvClient supplies the channel catalog used for
// broadcast-to-channel matching.
func New(baseURL string, tvClient *tv.Client) *Client {

	if baseURL == "" {

		baseURL = os.Getenv("TV_BASE_URL")

	}

	if baseURL == "" {

		baseURL = defaultBaseURL

	}

	return &Client{

		baseURL: strings.TrimRight(baseURL, "/"),
		server:  defaultServer,

		httpClient: &http.Client{Timeout: 30 * time.Second},
		tvClient:   tvClient,
	}

}

func (c *Client) get(rawURL string) (*http.Response, error) {

	request, err := http.NewRequest(http.MethodGet, rawURL, nil)

	if err != nil {

		return nil, err

	}

	request.Header.Set("User-Agent", browserUA)
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")

	return c.httpClient.Do(request)

}

// Matches returns current and upcoming sports matches, matched to channels
// where a broadcaster could be identified. Cached for matchesTTL.
func (c *Client) Matches() ([]Match, error) {

	if cached, ok := c.cachedMatches(); ok {

		return cached, nil

	}

	return c.refresh()

}

func (c *Client) cachedMatches() ([]Match, bool) {

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.matches == nil || time.Since(c.matchesAt) > matchesTTL {

		return nil, false

	}

	return append([]Match(nil), c.matches...), true

}

func (c *Client) refresh() ([]Match, error) {

	raw, err := c.fetchRawMatches()

	if err != nil {

		return nil, err

	}

	matches := make([]Match, 0, len(raw))

	for _, m := range raw {

		matches = append(matches, m.toMatch())

	}

	sortMatches(matches)

	if catalog, err := c.tvClient.ListChannels(); err == nil {

		c.matchChannels(matches, catalog)

	}

	c.mu.Lock()
	c.matches = matches
	c.matchesAt = time.Now()
	c.mu.Unlock()

	return append([]Match(nil), matches...), nil

}

func (c *Client) fetchRawMatches() ([]rawMatch, error) {

	url := fmt.Sprintf("%s/api/get-matches?server=%s&type=both", c.baseURL, c.server)

	response, err := c.get(url)

	if err != nil {

		return nil, fmt.Errorf("sports: fetch matches: %w", err)

	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {

		return nil, fmt.Errorf("sports: read matches response: %w", err)

	}

	if response.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("sports: fetch matches: status %d", response.StatusCode)

	}

	var parsed getMatchesResponse

	if err := json.Unmarshal(body, &parsed); err != nil {

		return nil, fmt.Errorf("sports: decode matches response: %w", err)

	}

	if !parsed.Success {

		return nil, fmt.Errorf("sports: matches response reported failure")

	}

	seen := make(map[string]rawMatch)

	for _, group := range [][]rawMatch{parsed.All, parsed.Live, parsed.NonLive} {

		for _, m := range group {

			if _, exists := seen[m.ID]; !exists {

				seen[m.ID] = m

			}

		}

	}

	out := make([]rawMatch, 0, len(seen))

	for _, m := range seen {

		out = append(out, m)

	}

	return out, nil

}

const (
	bucketLive = iota
	bucketUpcoming
	bucketPast
)

func bucketFor(m Match, now time.Time) int {

	if m.Live {

		return bucketLive

	}

	if m.StartTime.After(now) {

		return bucketUpcoming

	}

	return bucketPast

}

func sortMatches(matches []Match) {

	now := time.Now()

	sort.SliceStable(matches, func(i, j int) bool {

		bi, bj := bucketFor(matches[i], now), bucketFor(matches[j], now)

		if bi != bj {

			return bi < bj

		}

		switch bi {

		case bucketUpcoming:

			return matches[i].StartTime.Before(matches[j].StartTime)

		case bucketPast:

			return matches[i].StartTime.After(matches[j].StartTime)

		default:

			return matches[i].StartTime.After(matches[j].StartTime)

		}

	})

}
