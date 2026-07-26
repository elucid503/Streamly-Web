package source

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// NTV is the ntv.cx / cdnlive aggregator listed on FMHY.
// Catalog is name-matched; streams are extracted from player pages.

type ntvProvider struct {

	client *http.Client
	baseURL string

	mu sync.Mutex
	channels []ntvChannel
	fetchedAt time.Time

}

type ntvChannel struct {

	ID string `json:"channel_id"`
	Name string `json:"channel_name"`
	Code string `json:"channel_code"`
	PlayerURL string `json:"channel_url"`
	Server string `json:"server"`

}

type ntvChannelsResponse struct {

	Success bool `json:"success"`
	Channels []ntvChannel `json:"channels"`

}

var (
	ntvVarDeclRE = regexp.MustCompile(`var\s+(\w+)\s*=\s*'([^']*)';`)
	ntvChainRE = regexp.MustCompile(`var\s+\w+\s*=\s*((?:\w+\(\w+\)\s*\+\s*)*\w+\(\w+\))\s*;`)
	ntvCallRE = regexp.MustCompile(`\w+\((\w+)\)`)
)

// NewNTV builds the NTV source provider.
func NewNTV() Provider {

	return &ntvProvider{

		client: newHTTPClient(25 * time.Second),
		baseURL: "https://ntv.cx",

	}

}

func (p *ntvProvider) Name() string {

	return "ntv"

}

func (p *ntvProvider) Resolve(ctx context.Context, req Request) (Stream, error) {

	channels, err := p.listChannels(ctx)

	if err != nil {

		return Stream{}, err

	}

	ch, ok := p.matchChannel(req, channels)

	if !ok {

		return Stream{}, fmt.Errorf("ntv: no channel match for %q", req.Name)

	}

	streamURL, err := p.resolvePlayer(ctx, ch.PlayerURL)

	if err != nil {

		return Stream{}, err

	}

	headers := map[string]string{

		"Referer": "https://cdnlivetv.tv/",
		"Origin": "https://cdnlivetv.tv",
		"User-Agent": browserUA,

	}

	// Only accept playable playlists — NTV often returns tokenized URLs that 502.
	if !verifyPlaylist(ctx, p.client, streamURL, headers) {

		// Retry without referer (some edges accept bare requests).
		if !verifyPlaylist(ctx, p.client, streamURL, map[string]string{"User-Agent": browserUA}) {

			return Stream{}, fmt.Errorf("ntv: playlist not playable for %q", ch.Name)

		}

		headers = map[string]string{"User-Agent": browserUA}

	}

	return Stream{

		URL: streamURL,
		IsHLS: true,
		Headers: headers,
		Provider: p.Name(),

	}, nil

}

func (p *ntvProvider) listChannels(ctx context.Context) ([]ntvChannel, error) {

	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.fetchedAt) < 30*time.Minute && len(p.channels) > 0 {

		return p.channels, nil

	}

	body, status, err := getText(ctx, p.client, p.baseURL+"/api/get-channels", map[string]string{
		"Accept": "application/json",
		"Accept-Language": "en-US,en;q=0.9",
	})

	if err != nil {

		return nil, fmt.Errorf("ntv: fetch channels: %w", err)

	}

	if status != http.StatusOK {

		return nil, fmt.Errorf("ntv: fetch channels: status %d", status)

	}

	var parsed ntvChannelsResponse

	if err := json.Unmarshal([]byte(body), &parsed); err != nil {

		return nil, fmt.Errorf("ntv: decode channels: %w", err)

	}

	if !parsed.Success {

		return nil, fmt.Errorf("ntv: channels response reported failure")

	}

	// Prefer cdnlive — historically the only ntv server that resolved cleanly.
	filtered := make([]ntvChannel, 0, len(parsed.Channels))

	for _, ch := range parsed.Channels {

		if ch.Server == "cdnlive" || ch.Server == "" {

			filtered = append(filtered, ch)

		}

	}

	if len(filtered) == 0 {

		filtered = parsed.Channels

	}

	p.channels = filtered
	p.fetchedAt = time.Now()

	return p.channels, nil

}

func (p *ntvProvider) matchChannel(req Request, channels []ntvChannel) (ntvChannel, bool) {

	bestIdx := -1
	bestScore := 0

	for i, ch := range channels {

		score := matchScore(req, ch.Name)

		if score > bestScore {

			bestScore = score
			bestIdx = i

		}

	}

	if bestIdx < 0 || bestScore < 70 {

		return ntvChannel{}, false

	}

	return channels[bestIdx], true

}

func (p *ntvProvider) resolvePlayer(ctx context.Context, playerURL string) (string, error) {

	playerURL = strings.TrimSpace(playerURL)

	if playerURL == "" {

		return "", fmt.Errorf("ntv: empty player url")

	}

	body, status, err := getText(ctx, p.client, playerURL, map[string]string{
		"Accept-Language": "en-US,en;q=0.9",
	})

	if err != nil {

		return "", fmt.Errorf("ntv: fetch player: %w", err)

	}

	if status != http.StatusOK {

		return "", fmt.Errorf("ntv: fetch player: status %d", status)

	}

	return extractNTVStreamURL(body)

}

func extractNTVStreamURL(html string) (string, error) {

	literals := make(map[string]string)

	for _, match := range ntvVarDeclRE.FindAllStringSubmatch(html, -1) {

		literals[match[1]] = match[2]

	}

	if len(literals) == 0 {

		return "", fmt.Errorf("ntv: no base64 literals in player page")

	}

	var best []string

	for _, chain := range ntvChainRE.FindAllStringSubmatch(html, -1) {

		names := make([]string, 0)
		ok := true

		for _, call := range ntvCallRE.FindAllStringSubmatch(chain[1], -1) {

			if _, exists := literals[call[1]]; !exists {

				ok = false
				break

			}

			names = append(names, call[1])

		}

		if ok && len(names) > len(best) {

			best = names

		}

	}

	if len(best) == 0 {

		return "", fmt.Errorf("ntv: no stream url expression in player page")

	}

	var builder strings.Builder

	for _, name := range best {

		decoded, err := decodeURLSafeBase64(literals[name])

		if err != nil {

			return "", fmt.Errorf("ntv: decode fragment: %w", err)

		}

		builder.Write(decoded)

	}

	streamURL := builder.String()

	if !strings.HasPrefix(streamURL, "http://") && !strings.HasPrefix(streamURL, "https://") {

		return "", fmt.Errorf("ntv: decoded stream url invalid: %q", streamURL)

	}

	return streamURL, nil

}

func decodeURLSafeBase64(value string) ([]byte, error) {

	value = strings.ReplaceAll(value, "-", "+")
	value = strings.ReplaceAll(value, "_", "/")

	for len(value)%4 != 0 {

		value += "="

	}

	return base64.StdEncoding.DecodeString(value)

}
