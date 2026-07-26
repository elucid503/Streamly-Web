package source

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DaddyLive (dlhd.st) — FMHY starred Live TV / Sports source.
// Channel grid → stream embed → base64 HLS URL with player-host referer.

type daddyLiveProvider struct {

	client *http.Client
	baseURL string

	mu sync.Mutex
	channels map[string]int // normalized name -> id
	names []string
	fetchedAt time.Time

}

var (
	daddyCardRE = regexp.MustCompile(`(?is)<a\s+class="card"\s+href="/watch\.php\?id=(\d+)"\s+data-title="([^"]+)"`)
	daddyIFrameRE = regexp.MustCompile(`(?i)<iframe[^>]+src=["'](https?://[^"']*premiumtv[^"']+)["']`)
	daddyAtobRE = regexp.MustCompile(`(?i)source\s*:\s*window\.atob\(\s*'([A-Za-z0-9+/=]+)'\s*\)`)
	daddyAtobRE2 = regexp.MustCompile(`(?i)atob\(\s*'([A-Za-z0-9+/=]+)'\s*\)`)
)

// NewDaddyLive builds the DaddyLive source provider.
func NewDaddyLive() Provider {

	return &daddyLiveProvider{

		client: newHTTPClient(25 * time.Second),
		baseURL: "https://dlhd.st",

	}

}

func (p *daddyLiveProvider) Name() string {

	return "daddylive"

}

func (p *daddyLiveProvider) Resolve(ctx context.Context, req Request) (Stream, error) {

	id, name, err := p.match(ctx, req)

	if err != nil {

		return Stream{}, err

	}

	streamURL, referer, err := p.resolveStream(ctx, id)

	if err != nil {

		return Stream{}, fmt.Errorf("daddylive: resolve %q: %w", name, err)

	}

	headers := map[string]string{

		"User-Agent": browserUA,
		"Referer": referer,
		"Origin": originOf(referer),

	}

	if !verifyPlaylist(ctx, p.client, streamURL, headers) {

		return Stream{}, fmt.Errorf("daddylive: playlist not playable for %q", name)

	}

	return Stream{

		URL: streamURL,
		IsHLS: true,
		Headers: headers,
		Provider: p.Name(),

	}, nil

}

func (p *daddyLiveProvider) match(ctx context.Context, req Request) (int, string, error) {

	if err := p.ensureIndex(ctx); err != nil {

		return 0, "", err

	}

	p.mu.Lock()
	defer p.mu.Unlock()

	bestName, score := bestMatch(req, p.names, 70)

	if bestName == "" {

		return 0, "", fmt.Errorf("daddylive: no channel match for %q", req.Name)

	}

	id, ok := p.channels[normalizeName(bestName)]

	if !ok {

		// bestName is display form; try normalized map keys.
		for n, i := range p.channels {

			if matchScore(req, n) == score && matchScore(Request{Name: bestName}, n) >= 90 {

				return i, bestName, nil

			}

		}

		return 0, "", fmt.Errorf("daddylive: match index miss for %q", bestName)

	}

	return id, bestName, nil

}

func (p *daddyLiveProvider) ensureIndex(ctx context.Context) error {

	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.fetchedAt) < 45*time.Minute && len(p.channels) > 0 {

		return nil

	}

	body, status, err := getText(ctx, p.client, p.baseURL+"/24-7-channels.php", map[string]string{
		"Accept": "text/html",
		"Referer": p.baseURL + "/",
	})

	if err != nil {

		return fmt.Errorf("daddylive: fetch channel list: %w", err)

	}

	if status != http.StatusOK {

		return fmt.Errorf("daddylive: channel list status %d", status)

	}

	matches := daddyCardRE.FindAllStringSubmatch(body, -1)

	if len(matches) == 0 {

		return fmt.Errorf("daddylive: no channels parsed")

	}

	channels := make(map[string]int, len(matches))
	names := make([]string, 0, len(matches))

	for _, m := range matches {

		id := atoi(m[1])
		title := htmlUnescape(m[2])

		if id == 0 || title == "" {

			continue

		}

		key := normalizeName(title)

		if key == "" {

			continue

		}

		// Prefer lower ids when duplicates (often cleaner US feeds).
		if existing, ok := channels[key]; ok && existing <= id {

			continue

		}

		channels[key] = id
		names = append(names, title)

	}

	p.channels = channels
	p.names = names
	p.fetchedAt = time.Now()

	return nil

}

func (p *daddyLiveProvider) resolveStream(ctx context.Context, id int) (streamURL, referer string, err error) {

	// 1) watch page → stream embed
	watchURL := fmt.Sprintf("%s/watch.php?id=%d", p.baseURL, id)

	watchBody, status, err := getText(ctx, p.client, watchURL, map[string]string{
		"Referer": p.baseURL + "/",
	})

	if err != nil {

		return "", "", err

	}

	if status != http.StatusOK {

		return "", "", fmt.Errorf("watch status %d", status)

	}

	// Prefer explicit stream iframe.
	streamEmbed := fmt.Sprintf("%s/stream/stream-%d.php", p.baseURL, id)

	if m := regexp.MustCompile(`(?i)src=["'](https?://[^"']*stream/stream-\d+\.php)["']`).FindStringSubmatch(watchBody); len(m) == 2 {

		streamEmbed = m[1]

	}

	// 2) stream embed → premiumtv player host
	embedBody, status, err := getText(ctx, p.client, streamEmbed, map[string]string{
		"Referer": watchURL,
	})

	if err != nil {

		return "", "", err

	}

	if status != http.StatusOK {

		return "", "", fmt.Errorf("stream embed status %d", status)

	}

	playerURL := ""

	if m := daddyIFrameRE.FindStringSubmatch(embedBody); len(m) == 2 {

		playerURL = m[1]

	}

	if playerURL == "" {

		// Fallback common path pattern if iframe host moves.
		playerURL = fmt.Sprintf("%s/premiumtv/daddy3.php?id=%d", p.baseURL, id)

	}

	// 3) player page → base64 HLS
	playerBody, status, err := getText(ctx, p.client, playerURL, map[string]string{
		"Referer": streamEmbed,
	})

	if err != nil {

		return "", "", err

	}

	if status != http.StatusOK {

		return "", "", fmt.Errorf("player status %d", status)

	}

	b64 := ""

	if m := daddyAtobRE.FindStringSubmatch(playerBody); len(m) == 2 {

		b64 = m[1]

	} else if m := daddyAtobRE2.FindStringSubmatch(playerBody); len(m) == 2 {

		b64 = m[1]

	}

	if b64 == "" {

		return "", "", fmt.Errorf("no atob stream in player page")

	}

	decoded, err := base64.StdEncoding.DecodeString(b64)

	if err != nil {

		return "", "", fmt.Errorf("decode stream: %w", err)

	}

	streamURL = strings.TrimSpace(string(decoded))

	if !strings.HasPrefix(streamURL, "http") {

		return "", "", fmt.Errorf("invalid stream url")

	}

	referer = originOf(playerURL) + "/"

	return streamURL, referer, nil

}

func originOf(raw string) string {

	u, err := url.Parse(raw)

	if err != nil || u.Scheme == "" || u.Host == "" {

		return "https://dlhd.st"

	}

	return u.Scheme + "://" + u.Host

}

func atoi(s string) int {

	n := 0

	for _, r := range s {

		if r < '0' || r > '9' {

			return 0

		}

		n = n*10 + int(r-'0')

	}

	return n

}

func htmlUnescape(s string) string {

	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&nbsp;", " ",
	)

	return replacer.Replace(s)

}
