package sports

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
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

// matchServers is the ntv.cx match-feed server list in preference order.
// Individual servers routinely go empty while others still carry fixtures;
// fetch walks this list until one returns matches.
var matchServers = []string{
	"kobra",
	"raptor",
	"falcon",
	"phoenix",
	"viper",
	"titan",
}

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

// activeServer returns the match server used for the last successful listing
// (and therefore the server segment in /watch/{server}/{id} URLs).
func (c *Client) activeServer() string {

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.server == "" {

		return defaultServer

	}

	return c.server

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

	c.enrichScores(matches)

	// Re-sort after score enrichment may flip Live flags.
	sortMatches(matches)

	c.mu.Lock()
	c.matches = matches
	c.matchesAt = time.Now()
	c.mu.Unlock()

	return append([]Match(nil), matches...), nil

}

// fetchRawMatches walks matchServers until one returns a non-empty feed.
// An empty feed is not fatal — ntv frequently parks fixtures on alternate
// servers while the site default still answers success:true with [].
// Transport failures abort the walk early: every server shares one origin.
func (c *Client) fetchRawMatches() ([]rawMatch, error) {

	var lastErr error
	var sawEmpty bool

	for _, server := range matchServers {

		raw, err := c.fetchRawMatchesFrom(server)

		if err != nil {

			lastErr = err
			log.Printf("[sports] matches fetch failed for server %s: %v", server, err)

			// Same host for all matchServers — dial/timeouts won't recover on retry.
			if isTransportError(err) {

				return nil, err

			}

			continue

		}

		if len(raw) == 0 {

			sawEmpty = true
			continue

		}

		c.mu.Lock()
		prev := c.server
		c.server = server
		c.mu.Unlock()

		if server != prev || server != defaultServer {

			log.Printf("[sports] using match server %s (%d matches)", server, len(raw))

		}

		return raw, nil

	}

	if lastErr != nil && !sawEmpty {

		return nil, lastErr

	}

	// Every reachable server returned an empty success payload.
	if lastErr != nil {

		log.Printf("[sports] all match servers empty or failed; last error: %v", lastErr)

	}

	return []rawMatch{}, nil

}

// isTransportError reports dial/timeout/network failures (not HTTP 4xx/5xx).
func isTransportError(err error) bool {

	if err == nil {

		return false

	}

	var netErr net.Error

	if errors.As(err, &netErr) {

		return true

	}

	var dnsErr *net.DNSError

	if errors.As(err, &dnsErr) {

		return true

	}

	var opErr *net.OpError

	if errors.As(err, &opErr) {

		return true

	}

	// http.Client wraps many transport failures as url.Error / plain strings.
	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "tls handshake timeout") ||
		strings.Contains(msg, "wsarecv") ||
		strings.Contains(msg, "connectex")

}

func (c *Client) fetchRawMatchesFrom(server string) ([]rawMatch, error) {

	url := fmt.Sprintf("%s/api/get-matches?server=%s&type=both", c.baseURL, server)

	response, err := c.get(url)

	if err != nil {

		return nil, fmt.Errorf("sports: fetch matches (%s): %w", server, err)

	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {

		return nil, fmt.Errorf("sports: read matches response (%s): %w", server, err)

	}

	if response.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("sports: fetch matches (%s): status %d", server, response.StatusCode)

	}

	var parsed getMatchesResponse

	if err := json.Unmarshal(body, &parsed); err != nil {

		return nil, fmt.Errorf("sports: decode matches response (%s): %w", server, err)

	}

	if !parsed.Success {

		return nil, fmt.Errorf("sports: matches response reported failure (%s)", server)

	}

	return mergeRawMatchGroups(parsed.Live, parsed.NonLive, parsed.All), nil

}

// mergeRawMatchGroups dedupes by id across live/nonLive/all. Prefer Live over
// All/NonLive — ntv's "all" payload always stamps live:false even for matches
// that also appear in the live list.
func mergeRawMatchGroups(groups ...[]rawMatch) []rawMatch {

	seen := make(map[string]rawMatch)

	for _, group := range groups {

		for _, m := range group {

			if existing, exists := seen[m.ID]; exists {

				if !existing.Live && m.Live {

					existing.Live = true
					seen[m.ID] = existing

				}

				continue

			}

			seen[m.ID] = m

		}

	}

	out := make([]rawMatch, 0, len(seen))

	for _, m := range seen {

		out = append(out, m)

	}

	return out

}

const (
	bucketLive = iota
	bucketUpcoming
	bucketPast
)

func bucketFor(m Match, now time.Time) int {

	if m.Live || m.Status == StatusIn {

		return bucketLive

	}

	if m.Status == StatusPost {

		return bucketPast

	}

	// Scored non-live fixtures are finished; keep them out of the upcoming bucket.
	if m.HomeScore != nil && m.AwayScore != nil && m.Status != StatusPre {

		return bucketPast

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
