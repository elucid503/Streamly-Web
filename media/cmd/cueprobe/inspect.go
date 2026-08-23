package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	inspectUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36"

	playlistLimit = 2 << 20
	segmentLimit = 768 << 10
)

var (
	m3u8InPageRE = regexp.MustCompile(`https?://[^"'\\\s<>]+?\.m3u8[^"'\\\s<>]*`)
	quotedM3U8RE = regexp.MustCompile(`["']([^"']+\.m3u8[^"']*)["']`)
)

type inspectHTTP struct {

	client *http.Client

}

func newInspectHTTP(timeout time.Duration) *inspectHTTP {

	if timeout <= 0 {

		timeout = 25 * time.Second

	}

	return &inspectHTTP{

		client: &http.Client{

			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {

				if len(via) >= 8 {

					return fmt.Errorf("too many redirects")

				}

				return nil

			},

		},

	}

}

func (h *inspectHTTP) get(ctx context.Context, raw string, headers map[string]string, limit int64) ([]byte, int, error) {

	if limit <= 0 {

		limit = playlistLimit

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)

	if err != nil {

		return nil, 0, err

	}

	req.Header.Set("User-Agent", inspectUA)
	req.Header.Set("Accept", "*/*")

	for k, v := range headers {

		if v != "" {

			req.Header.Set(k, v)

		}

	}

	resp, err := h.client.Do(req)

	if err != nil {

		return nil, 0, err

	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))

	if err != nil {

		return nil, resp.StatusCode, err

	}

	return body, resp.StatusCode, nil

}

func (h *inspectHTTP) getText(ctx context.Context, raw string, headers map[string]string) (string, int, error) {

	body, status, err := h.get(ctx, raw, headers, playlistLimit)

	return string(body), status, err

}

type cueReport struct {

	MasterURL string
	MediaURL string
	SegmentURL string

	MasterTags []string
	MediaTags []string
	SegmentHits []string

	ClosedCaptions bool
	HasMediaPlaylist bool

}

func (r cueReport) anyCue() bool {

	return hasStrongCue(r.MasterTags) || hasStrongCue(r.MediaTags) || hasStrongSegment(r.SegmentHits)

}

func (r cueReport) anyHint() bool {

	return r.anyCue() || hasHint(r.MasterTags) || hasHint(r.MediaTags) ||
		r.ClosedCaptions || hasCaptionSegment(r.SegmentHits)

}

func inspectHLS(ctx context.Context, h *inspectHTTP, playlistURL string, headers map[string]string) (cueReport, error) {

	var out cueReport

	out.MasterURL = playlistURL

	body, status, err := h.getText(ctx, playlistURL, headers)

	if err != nil {

		return out, err

	}

	if status < 200 || status >= 300 {

		return out, fmt.Errorf("playlist status %d", status)

	}

	if !strings.Contains(body, "#EXTM3U") {

		return out, fmt.Errorf("not an HLS playlist")

	}

	out.MasterTags = collectCueTags(body)
	out.ClosedCaptions = strings.Contains(body, "CLOSED-CAPTIONS") ||
		strings.Contains(body, `INSTREAM-ID="CC`)

	mediaURL := playlistURL

	if isMasterPlaylist(body) {

		variant, vErr := firstVariantURL(playlistURL, body)

		if vErr != nil {

			return out, vErr

		}

		mediaURL = variant
		out.MediaURL = variant

		mediaBody, mediaStatus, mediaErr := h.getText(ctx, mediaURL, headers)

		if mediaErr != nil {

			return out, mediaErr

		}

		if mediaStatus < 200 || mediaStatus >= 300 {

			return out, fmt.Errorf("media playlist status %d", mediaStatus)

		}

		if !strings.Contains(mediaBody, "#EXTM3U") {

			return out, fmt.Errorf("media playlist not HLS")

		}

		out.HasMediaPlaylist = true
		out.MediaTags = collectCueTags(mediaBody)
		out.ClosedCaptions = out.ClosedCaptions ||
			strings.Contains(mediaBody, "CLOSED-CAPTIONS") ||
			strings.Contains(mediaBody, `INSTREAM-ID="CC`)

		body = mediaBody

	} else {

		out.MediaURL = playlistURL
		out.HasMediaPlaylist = true
		out.MediaTags = out.MasterTags

	}

	segURL, segErr := firstSegmentURL(mediaURL, body)

	if segErr != nil {

		return out, nil

	}

	out.SegmentURL = segURL

	seg, segStatus, segGetErr := h.get(ctx, segURL, headers, segmentLimit)

	if segGetErr != nil || segStatus < 200 || segStatus >= 300 {

		return out, nil

	}

	out.SegmentHits = scanSegment(seg)

	return out, nil

}

func isMasterPlaylist(body string) bool {

	return strings.Contains(body, "#EXT-X-STREAM-INF")

}

func firstVariantURL(playlistURL, body string) (string, error) {

	lines := strings.Split(body, "\n")

	for i, line := range lines {

		if !strings.HasPrefix(strings.TrimSpace(line), "#EXT-X-STREAM-INF") {

			continue

		}

		for j := i + 1; j < len(lines); j++ {

			u := strings.TrimSpace(lines[j])

			if u == "" || strings.HasPrefix(u, "#") {

				continue

			}

			return resolvePlaylistURL(playlistURL, u)

		}

	}

	return "", fmt.Errorf("no variant playlist")

}

func firstSegmentURL(playlistURL, body string) (string, error) {

	u, err := nthSegmentURL(playlistURL, body, false)

	return u, err

}

func lastSegmentURL(playlistURL, body string) (string, error) {

	return nthSegmentURL(playlistURL, body, true)

}

func nthSegmentURL(playlistURL, body string, last bool) (string, error) {

	found := ""

	for _, line := range strings.Split(body, "\n") {

		u := strings.TrimSpace(line)

		if u == "" || strings.HasPrefix(u, "#") {

			continue

		}

		lower := strings.ToLower(u)

		if strings.Contains(lower, ".m3u8") {

			continue

		}

		resolved, err := resolvePlaylistURL(playlistURL, stripBeacon(u))

		if err != nil {

			continue

		}

		if !last {

			return resolved, nil

		}

		found = resolved

	}

	if found == "" {

		return "", fmt.Errorf("no media segment")

	}

	return found, nil

}

func stripBeacon(raw string) string {

	// Xumo wraps some segments in a beacon URL; the real media is in ?url=
	if !strings.Contains(raw, "hlsstream/v1/beacon") {

		return raw

	}

	u, err := url.Parse(raw)

	if err != nil {

		return raw

	}

	inner := u.Query().Get("url")

	if inner == "" {

		return raw

	}

	return inner

}

func resolvePlaylistURL(base, ref string) (string, error) {

	ref = strings.TrimSpace(ref)

	if ref == "" {

		return "", fmt.Errorf("empty url")

	}

	if strings.HasPrefix(ref, "//") {

		bu, err := url.Parse(base)

		if err != nil {

			return "", err

		}

		return bu.Scheme + ":" + ref, nil

	}

	ru, err := url.Parse(ref)

	if err != nil {

		return "", err

	}

	if ru.IsAbs() {

		return ru.String(), nil

	}

	bu, err := url.Parse(base)

	if err != nil {

		return "", err

	}

	return bu.ResolveReference(ru).String(), nil

}

func collectCueTags(body string) []string {

	var out []string

	seen := map[string]bool{}

	add := func(label string) {

		if !seen[label] {

			seen[label] = true
			out = append(out, label)

		}

	}

	for _, line := range strings.Split(body, "\n") {

		line = strings.TrimSpace(line)

		if !strings.HasPrefix(line, "#") {

			if strings.Contains(strings.ToLower(line), "eventtype=ad") {

				add("eventType=AD")

			}

			if strings.Contains(strings.ToLower(line), "eventtype=asset") {

				add("eventType=ASSET")

			}

			continue

		}

		upper := strings.ToUpper(line)

		switch {

		case strings.HasPrefix(upper, "#EXT-X-CUE-OUT"):
			add("EXT-X-CUE-OUT")
		case strings.HasPrefix(upper, "#EXT-X-CUE-IN"):
			add("EXT-X-CUE-IN")
		case strings.HasPrefix(upper, "#EXT-X-CUE-OUT-CONT"):
			add("EXT-X-CUE-OUT-CONT")
		case strings.HasPrefix(upper, "#EXT-OATCLS-SCTE35"):
			add("EXT-OATCLS-SCTE35")
		case strings.HasPrefix(upper, "#EXT-X-SCTE35"):
			add("EXT-X-SCTE35")
		case strings.HasPrefix(upper, "#EXT-X-SPLICEPOINT-SCTE35"):
			add("EXT-X-SPLICEPOINT-SCTE35")
		case strings.HasPrefix(upper, "#EXT-X-DATERANGE"):
			if strings.Contains(upper, "SCTE35") {

				add("EXT-X-DATERANGE+SCTE35")

			} else {

				add("EXT-X-DATERANGE")

			}
		case strings.HasPrefix(upper, "#EXT-X-ASSET"):
			add("EXT-X-ASSET")
		case strings.HasPrefix(upper, "#EXT-X-DISCONTINUITY"):
			add("EXT-X-DISCONTINUITY")
		case strings.HasPrefix(upper, "#EXT-X-PROGRAM-DATE-TIME"):
			add("EXT-X-PROGRAM-DATE-TIME")
		}

	}

	return out

}

func hasStrongCue(tags []string) bool {

	for _, t := range tags {

		switch t {

		case "EXT-X-CUE-OUT", "EXT-X-CUE-IN", "EXT-X-CUE-OUT-CONT",
			"EXT-OATCLS-SCTE35", "EXT-X-SCTE35", "EXT-X-SPLICEPOINT-SCTE35",
			"EXT-X-DATERANGE+SCTE35", "eventType=AD":

			return true

		}

	}

	return false

}

func hasHint(tags []string) bool {

	for _, t := range tags {

		switch t {

		case "EXT-X-DISCONTINUITY", "EXT-X-DATERANGE", "EXT-X-ASSET",
			"EXT-X-PROGRAM-DATE-TIME", "eventType=ASSET":

			return true

		}

	}

	return false

}

func hasStrongSegment(hits []string) bool {

	for _, h := range hits {

		if strings.Contains(h, "0x86") || strings.Contains(h, "table_id=0xFC") ||
			strings.Contains(h, "fMP4 emsg") {

			return true

		}

	}

	return false

}

func hasCaptionSegment(hits []string) bool {

	for _, h := range hits {

		if strings.Contains(h, "GA94") {

			return true

		}

	}

	return false

}

func extractM3U8s(page string) []string {

	seen := map[string]bool{}
	var out []string

	add := func(raw string) {

		raw = strings.ReplaceAll(raw, `\/`, "/")
		raw = strings.Trim(raw, `"'`)

		if strings.HasPrefix(raw, "//") {

			raw = "https:" + raw

		}

		if !strings.Contains(strings.ToLower(raw), ".m3u8") {

			return

		}

		if !strings.HasPrefix(raw, "http") {

			return

		}

		if seen[raw] {

			return

		}

		seen[raw] = true
		out = append(out, raw)

	}

	for _, m := range m3u8InPageRE.FindAllString(page, 12) {

		add(m)

	}

	for _, m := range quotedM3U8RE.FindAllStringSubmatch(page, 12) {

		if len(m) > 1 {

			add(m[1])

		}

	}

	return out

}
