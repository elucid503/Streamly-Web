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

// DaddyLive (dlive.sx, formerly dlhd.st) — FMHY starred Live TV / Sports source.
// Channel grid → player folders → iframe hops → HLS (streamUrl / atob).

type daddyLiveProvider struct {

	client *http.Client
	embedClient *http.Client
	baseURL string

	mu sync.Mutex
	channels map[string]int // normalized name -> id
	names []string
	fetchedAt time.Time

}

var (
	daddyBaseCandidates = []string{
		"https://dlive.sx",
		"https://dlhd.st",
		"https://dlstreams.st",
	}

	daddyPlayerFolders = []string{
		"player", "casting", "plus", "watch", "stream", "cast",
	}

	daddyCardRE = regexp.MustCompile(`(?is)<a\s+class="card"\s+([^>]+)>`)
	daddyWatchIDRE = regexp.MustCompile(`(?i)watch\.php\?id=(\d+)`)
	daddyTitleRE = regexp.MustCompile(`(?i)data-title="([^"]+)"`)
	daddyIFrameRE = regexp.MustCompile(`(?i)<iframe[^>]+src=["']([^"']+)["']`)
	daddyAtobRE = regexp.MustCompile(`(?i)source\s*:\s*window\.atob\(\s*'([A-Za-z0-9+/=]+)'\s*\)`)
	daddyAtobRE2 = regexp.MustCompile(`(?i)atob\(\s*'([A-Za-z0-9+/=]{16,})'\s*\)`)
	daddyStreamURLRE = regexp.MustCompile(`(?i)streamUrl\s*:\s*"((?:\\.|[^"\\])*)"`)
	daddyFileM3U8RE = regexp.MustCompile(`(?i)(?:file|src|source)\s*[:=]\s*["'](https?://[^"']+\.m3u8[^"']*)["']`)
	daddyBareM3U8RE = regexp.MustCompile(`https://[^"'<\s]+\.m3u8[^"'<\s]*`)
)

const daddyMaxEmbedHops = 6

// NewDaddyLive builds the DaddyLive source provider.
func NewDaddyLive() Provider {

	return &daddyLiveProvider{

		client: newHTTPClient(25 * time.Second),
		embedClient: newHTTPClient(10 * time.Second),
		baseURL: daddyBaseCandidates[0],

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

	if verifyPlaylist(ctx, p.client, streamURL, nil) {

		return Stream{

			URL: streamURL,
			IsHLS: true,
			Provider: p.Name(),

		}, nil

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

	var last error

	for _, base := range daddyBaseCandidates {

		body, status, err := getText(ctx, p.client, strings.TrimRight(base, "/")+"/24-7-channels.php", map[string]string{
			"Accept": "text/html",
			"Referer": strings.TrimRight(base, "/") + "/",
		})

		if err != nil {

			last = fmt.Errorf("daddylive: fetch channel list: %w", err)
			continue

		}

		if status != http.StatusOK {

			last = fmt.Errorf("daddylive: channel list status %d", status)
			continue

		}

		parsed := parseDaddyCards(body)

		if len(parsed) == 0 {

			last = fmt.Errorf("daddylive: no channels parsed")
			continue

		}

		channels := make(map[string]int, len(parsed))
		names := make([]string, 0, len(parsed))

		for _, card := range parsed {

			key := normalizeName(card.title)

			if key == "" {

				continue

			}

			// Prefer lower ids when duplicates (often cleaner US feeds).
			if existing, ok := channels[key]; ok && existing <= card.id {

				continue

			}

			channels[key] = card.id
			names = append(names, card.title)

		}

		if len(channels) == 0 {

			last = fmt.Errorf("daddylive: no channels parsed")
			continue

		}

		p.baseURL = strings.TrimRight(base, "/")
		p.channels = channels
		p.names = names
		p.fetchedAt = time.Now()

		return nil

	}

	if last != nil {

		return last

	}

	return fmt.Errorf("daddylive: no channels parsed")

}

func (p *daddyLiveProvider) resolveStream(ctx context.Context, id int) (streamURL, referer string, err error) {

	watchURL := fmt.Sprintf("%s/watch.php?id=%d", p.baseURL, id)

	_, _, _ = getText(ctx, p.client, watchURL, map[string]string{
		"Referer": p.baseURL + "/",
	})

	var last error

	for _, folder := range daddyPlayerFolders {

		pageURL := fmt.Sprintf("%s/%s/stream-%d.php", p.baseURL, folder, id)

		found, ref, hopErr := p.resolveFromPage(ctx, pageURL, watchURL, map[string]bool{})

		if hopErr != nil {

			last = hopErr
			continue

		}

		if found != "" {

			return found, ref, nil

		}

	}

	if last != nil {

		return "", "", last

	}

	return "", "", fmt.Errorf("no playable embed")

}

func (p *daddyLiveProvider) resolveFromPage(ctx context.Context, pageURL, referer string, visited map[string]bool) (string, string, error) {

	pageURL = strings.TrimSpace(pageURL)

	if pageURL == "" || visited[pageURL] || len(visited) >= daddyMaxEmbedHops {

		return "", "", nil

	}

	visited[pageURL] = true

	body, status, err := getText(ctx, p.embedClient, pageURL, map[string]string{
		"Referer": referer,
	})

	if err != nil {

		return "", "", err

	}

	if status != http.StatusOK {

		return "", "", fmt.Errorf("embed status %d", status)

	}

	if stream := extractDaddyPlayableURL(body); stream != "" {

		return stream, originOf(pageURL) + "/", nil

	}

	var last error

	for _, iframe := range daddyIFrames(body, pageURL) {

		if isSkippableEmbed(iframe) || visited[iframe] {

			continue

		}

		found, ref, hopErr := p.resolveFromPage(ctx, iframe, pageURL, visited)

		if hopErr != nil {

			last = hopErr
			continue

		}

		if found != "" {

			return found, ref, nil

		}

	}

	return "", "", last

}

func parseDaddyCards(body string) []struct {
	id int
	title string
} {

	blocks := daddyCardRE.FindAllStringSubmatch(body, -1)
	out := make([]struct {
		id int
		title string
	}, 0, len(blocks))

	for _, m := range blocks {

		attrs := m[1]
		idMatch := daddyWatchIDRE.FindStringSubmatch(attrs)
		titleMatch := daddyTitleRE.FindStringSubmatch(attrs)

		if len(idMatch) < 2 || len(titleMatch) < 2 {

			continue

		}

		id := atoi(idMatch[1])
		title := htmlUnescape(titleMatch[1])

		if id == 0 || title == "" {

			continue

		}

		out = append(out, struct {
			id int
			title string
		}{id: id, title: title})

	}

	return out

}

func extractDaddyPlayableURL(html string) string {

	if m := daddyStreamURLRE.FindStringSubmatch(html); len(m) == 2 {

		if u := decodeDaddyQuotedURL(m[1]); looksLikeHLS(u) || strings.HasPrefix(u, "http") {

			return u

		}

	}

	if u, err := extractNTVStreamURL(html); err == nil && strings.HasPrefix(u, "http") {

		return u

	}

	if m := daddyAtobRE.FindStringSubmatch(html); len(m) == 2 {

		if u := decodeDaddyAtob(m[1]); u != "" {

			return u

		}

	}

	if m := daddyAtobRE2.FindStringSubmatch(html); len(m) == 2 {

		if u := decodeDaddyAtob(m[1]); u != "" {

			return u

		}

	}

	if m := daddyFileM3U8RE.FindStringSubmatch(html); len(m) == 2 {

		if u := decodeDaddyQuotedURL(m[1]); u != "" {

			return u

		}

	}

	if m := daddyBareM3U8RE.FindString(html); m != "" {

		if u := decodeDaddyQuotedURL(m); u != "" {

			return u

		}

	}

	return ""

}

func decodeDaddyAtob(b64 string) string {

	decoded, err := base64.StdEncoding.DecodeString(b64)

	if err != nil {

		return ""

	}

	u := strings.TrimSpace(string(decoded))

	if !strings.HasPrefix(u, "http") {

		return ""

	}

	return u

}

func decodeDaddyQuotedURL(raw string) string {

	raw = strings.TrimSpace(htmlUnescape(raw))
	raw = strings.ReplaceAll(raw, `\/`, `/`)
	raw = strings.Trim(raw, `"'`)

	if !strings.HasPrefix(raw, "http") {

		return ""

	}

	return raw

}

func daddyIFrames(html, pageURL string) []string {

	matches := daddyIFrameRE.FindAllStringSubmatch(html, -1)
	out := make([]string, 0, len(matches))
	seen := map[string]bool{}

	for _, m := range matches {

		abs := absURL(pageURL, htmlUnescape(m[1]))

		if abs == "" || seen[abs] {

			continue

		}

		seen[abs] = true
		out = append(out, abs)

	}

	for i := 0; i < len(out); i++ {

		for j := i + 1; j < len(out); j++ {

			if daddyIFrameScore(out[j]) > daddyIFrameScore(out[i]) {

				out[i], out[j] = out[j], out[i]

			}

		}

	}

	return out

}

func daddyIFrameScore(raw string) int {

	host := strings.ToLower(raw)

	switch {

	case strings.Contains(host, "wideiptv"):

		return 50

	case strings.Contains(host, "cdnlivetv"):

		return 40

	case strings.Contains(host, "dlive.sx"), strings.Contains(host, "dlhd."):

		return 20

	case strings.Contains(host, "premiumtv"):

		return 5

	default:

		return 10

	}

}

func isSkippableEmbed(raw string) bool {

	host := strings.ToLower(raw)

	for _, needle := range []string{
		"assetrage", "tiestep", "popcdn", "adbpage", "histats",
		"doubleclick", "googlesyndication", "hubeamily", "trovesleepit",
		"fellfortunate", "piousshiners", "nanisms", "xads",
		"rocketstreams", "ksohls", "romponalis", "about:blank", "javascript:",
	} {

		if strings.Contains(host, needle) {

			return true

		}

	}

	return false

}

func absURL(base, ref string) string {

	ref = strings.TrimSpace(ref)

	if ref == "" {

		return ""

	}

	u, err := url.Parse(ref)

	if err != nil {

		return ""

	}

	if u.IsAbs() {

		return u.String()

	}

	b, err := url.Parse(base)

	if err != nil {

		return ""

	}

	return b.ResolveReference(u).String()

}

func originOf(raw string) string {

	u, err := url.Parse(raw)

	if err != nil || u.Scheme == "" || u.Host == "" {

		return ""

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
