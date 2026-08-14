package sports

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"mediakit/internal/live/catalog"
	"mediakit/internal/live/source"
)

const matchesTTL = 2 * time.Minute

// Client fetches sports fixtures from scoreboard APIs (ESPN) and optionally
// soft-links them to catalog channels by name. It does not talk to stream providers.
type Client struct {
	httpClient *http.Client
	catalog    *catalog.Client
	resolver   *source.Resolver

	mu        sync.RWMutex
	matches   []Match
	matchesAt time.Time
}

const sportsChannelPrefix = "sports-ntv-"

// EncodeChannelID creates the opaque channel identity used for an NTV
// team-named sports feed.
func EncodeChannelID(name string) string {

	return sportsChannelPrefix + base64.RawURLEncoding.EncodeToString([]byte(name))

}

// DecodeChannelID validates and decodes an NTV sports channel identity.
func DecodeChannelID(id string) (string, bool) {

	if !strings.HasPrefix(id, sportsChannelPrefix) {
		return "", false
	}

	raw := strings.TrimPrefix(id, sportsChannelPrefix)
	if raw == "" || len(raw) > 256 {
		return "", false
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", false
	}

	name := strings.TrimSpace(string(decoded))
	if name == "" || len(name) > 120 || EncodeChannelID(name) != id {
		return "", false
	}

	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return "", false
		}
	}

	return name, true

}

// New builds a sports Client. catalog and resolver may be nil.
func New(cat *catalog.Client, resolver *source.Resolver) *Client {

	return &Client{

		httpClient: &http.Client{Timeout: espnFetchTimeout},
		catalog:    cat,
		resolver:   resolver,
	}

}

// Matches returns live and upcoming fixtures, cached briefly.
func (c *Client) Matches() ([]Match, error) {

	c.mu.RLock()

	if time.Since(c.matchesAt) < matchesTTL && c.matches != nil {

		out := append([]Match(nil), c.matches...)
		c.mu.RUnlock()

		return out, nil

	}

	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(c.matchesAt) < matchesTTL && c.matches != nil {

		return append([]Match(nil), c.matches...), nil

	}

	matches, err := fetchAllLeagues(c.httpClient)

	if err != nil {

		if c.matches != nil {

			log.Printf("[live/sports] refresh failed, serving stale: %v", err)
			return append([]Match(nil), c.matches...), nil

		}

		return nil, err

	}

	c.attachChannels(matches)
	sortMatches(matches)

	c.matches = matches
	c.matchesAt = time.Now()

	return append([]Match(nil), matches...), nil

}

func (c *Client) attachChannels(matches []Match) {

	var cat *catalog.Catalog
	if c.catalog != nil {
		loaded, err := c.catalog.List()
		if err == nil {
			cat = loaded
		}
	}

	for i := range matches {

		if cat != nil {
			matches[i].Channel = matchChannel(matches[i], cat)
		}
		if matches[i].Channel == nil {
			matches[i].Channel = c.matchNTVTeam(matches[i])
		}

	}

}

func (c *Client) matchNTVTeam(match Match) *MatchedChannel {

	if c.resolver == nil {
		return nil
	}

	// Prefer the home feed, then the away feed. NTV exposes many team-named
	// channels rather than event or broadcast-network names.
	for _, team := range []*Team{match.HomeTeam, match.AwayTeam} {
		if team == nil || team.Name == "" {
			continue
		}

		req := source.Request{Name: team.Name}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ok := c.resolver.Matches(ctx, req, "s2")
		cancel()
		if !ok {
			continue
		}

		return &MatchedChannel{ChannelID: EncodeChannelID(team.Name), Name: team.Name, Logo: team.Logo}
	}

	return nil

}

func sortMatches(matches []Match) {

	rank := func(m Match) int {

		switch m.Status {

		case StatusIn:

			return 0

		case StatusPre:

			return 1

		case StatusPost:

			return 2

		}

		if m.Live {

			return 0

		}

		return 1

	}

	sort.SliceStable(matches, func(i, j int) bool {

		ri, rj := rank(matches[i]), rank(matches[j])

		if ri != rj {

			return ri < rj

		}

		return matches[i].StartTime.Before(matches[j].StartTime)

	})

}
