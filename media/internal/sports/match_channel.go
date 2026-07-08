package sports

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"

	"mediakit/internal/tv"
)

// streamSuffixRE matches the trailing "Stream 1 [HD]"-style token in a
// ntv.cx source-select option label so it can be stripped before looking for
// a broadcaster name.
var streamSuffixRE = regexp.MustCompile(`(?i)^stream\s+\d+(\s*\[.*\])?$`)

const matchChannelConcurrency = 8

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

// matchChannelFor resolves the 24/7 channel a match should open on. It tries,
// in order: (1) a channel named after the home or away team — ntv.cx's own
// catalog includes team-branded channels for leagues like MLB, which is both
// the highest-precision match and avoids a network round-trip; (2) the
// broadcaster name parsed from the match's watch page (e.g. "Sportsnet LA"),
// matched against the catalog by name. Returns nil when neither yields a
// channel — not every match is simulcast somewhere we can identify.
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

	candidate, err := c.broadcasterLabel(m.ID)

	if err != nil || candidate == "" {

		return nil

	}

	hits := catalog.Search(candidate, 1)

	if len(hits) == 0 {

		return nil

	}

	return matchedChannelFromChannel(hits[0])

}

func matchedChannelFromChannel(ch tv.Channel) *MatchedChannel {

	return &MatchedChannel{ChannelID: ch.ID, Name: ch.Name, Logo: ch.Logo}

}

// broadcasterLabel fetches a match's watch page and returns the first
// identifiable broadcaster name from its source-select option labels (e.g.
// "Server Kobra - ADMIN - English - Sportsnet LA - Stream 1 [HD]").
func (c *Client) broadcasterLabel(matchID string) (string, error) {

	watchURL := fmt.Sprintf("%s/watch/%s/%s", c.baseURL, c.server, matchID)

	response, err := c.get(watchURL)

	if err != nil {

		return "", fmt.Errorf("sports: fetch watch page: %w", err)

	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {

		return "", fmt.Errorf("sports: watch page status %d", response.StatusCode)

	}

	doc, err := goquery.NewDocumentFromReader(response.Body)

	if err != nil {

		return "", fmt.Errorf("sports: parse watch page: %w", err)

	}

	var candidate string

	doc.Find("#sourceSelect option").EachWithBreak(func(_ int, sel *goquery.Selection) bool {

		if name := broadcasterFromLabel(sel.Text()); name != "" {

			candidate = name
			return false

		}

		return true

	})

	return candidate, nil

}

// broadcasterFromLabel extracts the broadcaster name from a source-select
// option label. Labels follow "Server {name} - {SOURCE} - {Language} -
// {Broadcaster} - Stream N [HD]"; the language and broadcaster segments are
// each optional, so a broadcaster name is only present when exactly two
// segments remain after dropping the server/source prefix and stream suffix.
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
