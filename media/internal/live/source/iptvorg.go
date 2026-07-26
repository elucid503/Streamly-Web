package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// IPTVOrg is the open stream index from iptv-org (used by EasyWebTV / FMHY IPTV tools).
// Matches catalog channel IDs directly (e.g. ESPN.us) and verifies playlists live.

type iptvOrgProvider struct {

	client *http.Client

	mu sync.Mutex
	byChannel map[string][]iptvOrgStream
	fetchedAt time.Time

}

type iptvOrgStream struct {

	Channel string `json:"channel"`
	Feed string `json:"feed"`
	Title string `json:"title"`
	URL string `json:"url"`
	Quality string `json:"quality"`
	UserAgent string `json:"user_agent"`
	Referrer string `json:"referrer"`

}

// NewIPTVOrg builds the iptv-org streams provider.
func NewIPTVOrg() Provider {

	return &iptvOrgProvider{

		client: newHTTPClient(45 * time.Second),

	}

}

func (p *iptvOrgProvider) Name() string {

	return "iptvorg"

}

func (p *iptvOrgProvider) Resolve(ctx context.Context, req Request) (Stream, error) {

	if err := p.ensureIndex(ctx); err != nil {

		return Stream{}, err

	}

	candidates := p.candidates(req)

	if len(candidates) == 0 {

		return Stream{}, fmt.Errorf("iptvorg: no streams for %q", firstNonEmpty(req.ChannelID, req.Name))

	}

	// Prefer higher quality labels and https URLs.
	sortIPTVStreams(candidates)

	var last error

	for _, s := range candidates {

		headers := map[string]string{

			"User-Agent": firstNonEmpty(s.UserAgent, browserUA),
			"Accept": "*/*",

		}

		if s.Referrer != "" {

			headers["Referer"] = s.Referrer
			headers["Origin"] = originOf(s.Referrer)

		}

		if !verifyPlaylist(ctx, p.client, s.URL, headers) {

			last = fmt.Errorf("dead stream")
			continue

		}

		return Stream{

			URL: s.URL,
			IsHLS: true,
			Headers: headers,
			Provider: p.Name(),

		}, nil

	}

	if last != nil {

		return Stream{}, fmt.Errorf("iptvorg: no playable stream for %q", firstNonEmpty(req.ChannelID, req.Name))

	}

	return Stream{}, fmt.Errorf("iptvorg: no playable stream for %q", firstNonEmpty(req.ChannelID, req.Name))

}

func (p *iptvOrgProvider) candidates(req Request) []iptvOrgStream {

	p.mu.Lock()
	defer p.mu.Unlock()

	var out []iptvOrgStream

	if req.ChannelID != "" {

		out = append(out, p.byChannel[req.ChannelID]...)

	}

	// Case-insensitive id fallback.
	if len(out) == 0 && req.ChannelID != "" {

		for id, streams := range p.byChannel {

			if strings.EqualFold(id, req.ChannelID) {

				out = append(out, streams...)
				break

			}

		}

	}

	return out

}

func (p *iptvOrgProvider) ensureIndex(ctx context.Context) error {

	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.fetchedAt) < 2*time.Hour && len(p.byChannel) > 0 {

		return nil

	}

	body, status, err := getText(ctx, p.client, "https://iptv-org.github.io/api/streams.json", map[string]string{
		"Accept": "application/json",
	})

	if err != nil {

		return fmt.Errorf("iptvorg: fetch streams: %w", err)

	}

	if status != http.StatusOK {

		return fmt.Errorf("iptvorg: streams status %d", status)

	}

	var streams []iptvOrgStream

	if err := json.Unmarshal([]byte(body), &streams); err != nil {

		return fmt.Errorf("iptvorg: decode: %w", err)

	}

	byChannel := make(map[string][]iptvOrgStream, 4096)

	for _, s := range streams {

		s.Channel = strings.TrimSpace(s.Channel)
		s.URL = strings.TrimSpace(s.URL)

		if s.Channel == "" || s.URL == "" {

			continue

		}

		// Prefer https; keep http as last resort.
		byChannel[s.Channel] = append(byChannel[s.Channel], s)

	}

	if len(byChannel) == 0 {

		return fmt.Errorf("iptvorg: empty stream index")

	}

	p.byChannel = byChannel
	p.fetchedAt = time.Now()

	return nil

}

func sortIPTVStreams(streams []iptvOrgStream) {

	// Stable quality preference without importing sort.Slice noise: bubble small N.
	score := func(s iptvOrgStream) int {

		n := 0

		if strings.HasPrefix(strings.ToLower(s.URL), "https://") {

			n += 50

		}

		q := strings.ToLower(s.Quality)

		switch {

		case strings.Contains(q, "1080"):

			n += 30

		case strings.Contains(q, "720"):

			n += 20

		case strings.Contains(q, "480"):

			n += 10

		}

		return n

	}

	for i := 0; i < len(streams); i++ {

		for j := i + 1; j < len(streams); j++ {

			if score(streams[j]) > score(streams[i]) {

				streams[i], streams[j] = streams[j], streams[i]

			}

		}

	}

}
