package sports

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/PuerkitoBio/goquery"

	"mediakit/internal/tv"
)

// streamSuffixRE matches the trailing "Stream 1 [HD]"-style token in a ntv.cx source-select option label so it can be stripped before looking for a broadcaster name.
var streamSuffixRE = regexp.MustCompile(`(?i)^stream\s+\d+(\s*\[.*\])?$`)

const matchChannelConcurrency = 8

// Minimum score from channelNameMatchScore before a broadcaster is accepted.
const minBroadcasterScore = 70

func (c *Client) matchChannels(matches []Match, catalog *tv.ChannelCatalog) {

	sem := make(chan struct{}, matchChannelConcurrency)
	var wg sync.WaitGroup

	for i := range matches {

		wg.Add(1)
		sem <- struct{}{}

		go func(i int) {

			defer wg.Done()
			defer func() { <-sem }()

			matches[i].Channel = c.matchChannelFor(&matches[i], catalog)

		}(i)

	}

	wg.Wait()

}

func (c *Client) matchChannelFor(m *Match, catalog *tv.ChannelCatalog) *MatchedChannel {

	if m.HomeTeam != nil {

		if ch, ok := catalog.FindByExactName(m.HomeTeam.Name); ok {

			return matchedChannelFromChannel(ch)

		}

	}

	if m.AwayTeam != nil {

		if ch, ok := catalog.FindByExactName(m.AwayTeam.Name); ok {

			return matchedChannelFromChannel(ch)

		}

	}

	candidates, err := c.broadcasterLabels(m.ID)

	if err != nil || len(candidates) == 0 {

		return nil

	}

	return bestChannelForBroadcasters(catalog, candidates)

}

func matchedChannelFromChannel(ch tv.Channel) *MatchedChannel {

	return &MatchedChannel{ChannelID: ch.ID, Name: ch.Name, Logo: ch.Logo}

}

// broadcasterLabels fetches a match's watch page and returns every identifiable broadcaster name from its source-select option labels, in page order.
func (c *Client) broadcasterLabels(matchID string) ([]string, error) {

	watchURL := fmt.Sprintf("%s/watch/%s/%s", c.baseURL, c.server, matchID)

	response, err := c.get(watchURL)

	if err != nil {

		return nil, fmt.Errorf("sports: fetch watch page: %w", err)

	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("sports: watch page status %d", response.StatusCode)

	}

	doc, err := goquery.NewDocumentFromReader(response.Body)

	if err != nil {

		return nil, fmt.Errorf("sports: parse watch page: %w", err)

	}

	seen := make(map[string]struct{})
	var out []string

	doc.Find("#sourceSelect option").Each(func(_ int, sel *goquery.Selection) {

		for _, name := range expandBroadcasterName(broadcasterFromLabel(sel.Text())) {

			key := strings.ToLower(name)

			if _, exists := seen[key]; exists {

				continue

			}

			seen[key] = struct{}{}
			out = append(out, name)

		}

	})

	return out, nil

}

// expandBroadcasterName splits composite labels like "BBC/ITV" into individual candidates while keeping the original form first.
func expandBroadcasterName(name string) []string {

	name = strings.TrimSpace(name)

	if name == "" {

		return nil

	}

	out := []string{name}

	for _, sep := range []string{"/", "|", "&", "+"} {

		if !strings.Contains(name, sep) {

			continue

		}

		for _, part := range strings.Split(name, sep) {

			part = strings.TrimSpace(part)

			if part == "" || strings.EqualFold(part, name) {

				continue

			}

			out = append(out, part)

		}

	}

	return out

}

// bestChannelForBroadcasters picks the catalog channel that best matches any of the broadcaster labels, using token-aware scoring so short codes like "TSN" do not latch onto "SportsNet".
func bestChannelForBroadcasters(catalog *tv.ChannelCatalog, broadcasters []string) *MatchedChannel {

	if catalog == nil || len(broadcasters) == 0 {

		return nil

	}

	bestScore := 0
	var best tv.Channel

	for _, broadcaster := range broadcasters {

		if ch, ok := catalog.FindByExactName(broadcaster); ok {

			return matchedChannelFromChannel(ch)

		}

		for _, ch := range catalog.Channels {

			score := channelNameMatchScore(broadcaster, ch.Name)

			if score < minBroadcasterScore {

				continue

			}

			if score > bestScore || (score == bestScore && channelPreferred(ch, best)) {

				bestScore = score
				best = ch

			}

		}

	}

	if bestScore < minBroadcasterScore || best.ID == "" {

		return nil

	}

	return matchedChannelFromChannel(best)

}

// channelPreferred breaks ties: enriched (logo/metadata) first, then shorter name (prefer "Fox" over "Fox Sports 1" when scores tie), then alpha.
func channelPreferred(a, b tv.Channel) bool {

	if a.Enriched != b.Enriched {

		return a.Enriched

	}

	if len(a.Name) != len(b.Name) {

		return len(a.Name) < len(b.Name)

	}

	return strings.Compare(a.Name, b.Name) < 0

}

// channelNameMatchScore ranks how well a broadcaster label maps to a channel name. Higher is better; 0 means no usable match.
func channelNameMatchScore(query, name string) int {

	q := strings.ToLower(strings.TrimSpace(query))
	n := strings.ToLower(strings.TrimSpace(name))

	if q == "" || n == "" {

		return 0

	}

	if q == n {

		return 100

	}

	if strings.HasPrefix(n, q) {

		rest := n[len(q):]

		if rest == "" {

			return 100

		}

		if isNumericSuffix(rest) {

			return 90

		}

		if rest[0] == ' ' || rest[0] == '-' {

			return 75

		}

	}

	// Whole-token hit: "USA" in "CBS Sports Network USA", not "TSN" in "SportsNet".
	if hasNameToken(n, q) {

		return 85

	}

	// Long substring only — short codes false-positive too often.
	if len(q) >= 6 && strings.Contains(n, q) {

		return 50

	}

	return 0

}

func isNumericSuffix(s string) bool {

	s = strings.TrimSpace(s)

	if s == "" {

		return false

	}

	// Allow "1", " 1", "1 HD" style tails after a known brand stem.
	for i, r := range s {

		if unicode.IsDigit(r) {

			continue

		}

		if i == 0 {

			return false

		}

		// Digits then separator is fine ("1 HD"); non-digit first is not.
		return r == ' ' || r == '-'

	}

	return true

}

func hasNameToken(name, token string) bool {

	if token == "" {

		return false

	}

	return slices.Contains(strings.FieldsFunc(name, func(r rune) bool {

	 return !unicode.IsLetter(r) && !unicode.IsDigit(r)

	}), token)

}

func broadcasterFromLabel(label string) string {

	parts := strings.Split(strings.TrimSpace(label), " - ")

	for i := range parts {

		parts[i] = strings.TrimSpace(parts[i])

	}

	if len(parts) < 2 {

		return ""

	}

	rest := parts[2:]

	if len(rest) > 0 && streamSuffixRE.MatchString(rest[len(rest)-1]) {

		rest = rest[:len(rest)-1]

	}

	if len(rest) == 2 && rest[1] != "" {

		return rest[1]

	}

	return ""

}
